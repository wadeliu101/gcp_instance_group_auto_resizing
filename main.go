package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/zclconf/go-cty/cty"
	"golang.org/x/oauth2"
	auth "golang.org/x/oauth2/google"
)

type PromqlResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct {
				GroupName string `json:"group_name"`
			} `json:"metric"`
			Value []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

type Tfvars struct {
	project_id   string
	group_name   string
	max_replicas int
}

var (
	project_id          = flag.String("project_id", "", "target project_id")
	group_name          = flag.String("group_name", "", "target instance group")
	max_replicas_list   = []int{5, 10, 20, 40, 80, 160, 320}
	terraform_file_path = "./terraform"
)

func main() {
	flag.Parse()
	token, err := get_oauth_token()
	if err != nil {
		fmt.Println(err)
		return
	}
	process_exist_number, err := query_current_status(token, *project_id, "count(rate(custom_googleapis_com:opencensus_process_exists{monitored_resource=\"gce_instance\",project_id=\""+*project_id+"\",group_name=\""+*group_name+"\"}[5m]) > 0) by (group_name)")
	if err != nil {
		fmt.Println(err)
		return
	}

	instance_exist_number, err := query_current_status(token, *project_id, "compute_googleapis_com:instance_group_size{project_id=\""+*project_id+"\",instance_group_name=\""+*group_name+"\"}")
	if err != nil {
		fmt.Println(err)
		return
	}
	if process_exist_number == instance_exist_number {
		next_instance_number := find_next_vm_size(instance_exist_number)
		terraform_exec(terraform_file_path, Tfvars{
			project_id:   *project_id,
			group_name:   *group_name,
			max_replicas: next_instance_number,
		})
		fmt.Println(process_exist_number, instance_exist_number, next_instance_number)
	} else if process_exist_number > instance_exist_number {
		// 因為指標延遲的關係，所以暫不處理
		fmt.Println(process_exist_number, instance_exist_number)
		return
	} else {
		next_instance_number := find_prev_vm_size(instance_exist_number)
		if next_instance_number < process_exist_number {
			fmt.Println(process_exist_number, instance_exist_number, next_instance_number)
			return
		}
		terraform_exec(terraform_file_path, Tfvars{
			project_id:   *project_id,
			group_name:   *group_name,
			max_replicas: next_instance_number,
		})
		fmt.Println(process_exist_number, instance_exist_number, next_instance_number)
	}
}

func get_oauth_token() (string, error) {
	var token *oauth2.Token
	ctx := context.Background()
	scopes := []string{
		"https://www.googleapis.com/auth/cloud-platform",
	}
	credentials, err := auth.FindDefaultCredentials(ctx, scopes...)
	if err != nil {
		return "", err
	}
	token, err = credentials.TokenSource.Token()
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}

func query_current_status(token string, project_id string, query string) (int, error) {
	payload := strings.NewReader("query=" + url.QueryEscape(query))
	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://monitoring.googleapis.com/v1/projects/"+project_id+"/location/global/prometheus/api/v1/query", payload)
	if err != nil {
		return 0, err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}
	var response PromqlResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return 0, err
	}
	if len(response.Data.Result) == 0 {
		return 0, nil
	}
	return strconv.Atoi(fmt.Sprintf("%v", response.Data.Result[0].Value[1]))
}

func find_next_vm_size(current_size int) int {
	for index, size := range max_replicas_list {
		if index+1 == len(max_replicas_list) {
			return current_size
		}
		if current_size < size {
			return size
		}
	}
	return current_size
}

func find_prev_vm_size(current_size int) int {
	for index, size := range max_replicas_list {
		if current_size <= size {
			switch index {
			case 0:
				return current_size
			default:
				return max_replicas_list[index-1]
			}
		}
	}
	return current_size
}

func terraform_exec(workingDir string, tfvars Tfvars) error {
	installer := &releases.ExactVersion{
		Product: product.Terraform,
		Version: version.Must(version.NewVersion("1.0.6")),
	}

	execPath, err := installer.Install(context.Background())
	if err != nil {
		log.Fatalf("error installing Terraform: %s", err)
	}

	tf, err := tfexec.NewTerraform(workingDir, execPath)
	if err != nil {
		log.Fatalf("error running NewTerraform: %s", err)
	}

	err = tf.Init(context.Background(), tfexec.Upgrade(true))
	if err != nil {
		log.Fatalf("error running Init: %s", err)
	}

	f := hclwrite.NewEmptyFile()
	rootBody := f.Body()

	rootBody.SetAttributeValue("project_id", cty.StringVal(tfvars.project_id))
	rootBody.SetAttributeValue("group_name", cty.StringVal(tfvars.group_name))
	rootBody.SetAttributeValue("max_replicas", cty.NumberIntVal(int64(tfvars.max_replicas)))

	tfvars_file, err := os.OpenFile(workingDir+"/terraform.tfvars", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)

	if err != nil {
		log.Fatalf("error open file: %s", err)
	}

	defer tfvars_file.Close()

	tfvars_file.Truncate(0)
	tfvars_file.Seek(0, 0)
	fmt.Fprintf(tfvars_file, "%v", string(f.Bytes()))

	err = tf.Apply(context.Background())

	if err != nil {
		return fmt.Errorf("error running Apply: %s", err)
	}

	state, err := tf.Show(context.Background())

	if err != nil {
		log.Fatalf("error running Show: %s", err)
	}

	fmt.Println(state.Values.Outputs)

	return nil
}
