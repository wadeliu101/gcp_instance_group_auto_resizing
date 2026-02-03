# gcp-scale-controller 🚀

**簡介**

一個用 Go 開發的專案，搭配 Terraform 用於在 GCP 上部署與管理資源（專案中包含 `main.go` 與 `terraform/` 目錄）。此 README 提供快速開始、建置、以及 Terraform 部署流程。

## 主要功能 🔍

本專案主要負責監測兩個 Prometheus 指標：`custom_googleapis_com:opencensus_process_exists`（以 count(rate(...)[5m]) 判定 process 是否存在）與 `compute_googleapis_com:instance_group_size`（instance group 的當前大小）。程式會比較這兩項指標，並決定將 instance group 增加或縮小至適當的數量（透過 Terraform 更新 `max_replicas`），以自動調整實例數量達到所需的運行狀態。

---

## 目錄 📋
- **需求**
- **快速開始**
- **建置與執行**
- **Terraform 使用**
- **設定說明**
- **貢獻**
- **授權**

---

## 需求 ✅
- Go (建議 >= 1.20)
- Terraform (建議 >= 1.4)
- gcloud CLI 或 GCP JSON service account 金鑰

> Tip: 在 macOS 上可使用 `brew install go terraform google-cloud-sdk` 安裝。

---

## 快速開始 💡
1. 取得程式碼：

```bash
git clone <repo-url>
cd gcp_instance_group_auto_resizing
```

2. 驗證 GCP 認證（擇一）：

- 使用 gcloud：

```bash
gcloud auth application-default login
```

- 或者使用 Service Account Key：

```bash
export GOOGLE_APPLICATION_CREDENTIALS="$HOME/path/to/key.json"
```

---

## 建置與執行 🔧
在開發或測試時可以直接用 `go run`：

```bash
go run main.go
```

或建置後執行：

```bash
go build -o gcp-scale-controller main.go
./gcp-scale-controller
```

### 執行參數與用法 ▶️

執行 `gcp-scale-controller` 時可透過 flag 傳入必要參數：

```bash
./gcp-scale-controller -project_id <GCP_PROJECT_ID> -group_name <GCP_INSTANCE_GROUP>
```

參數說明：
- `-project_id`：GCP 專案 ID（必填）
- `-group_name`：GCP instance group 名稱（必填）

範例：

```bash
./gcp-scale-controller -project_id my-gcp-project -group_name my-instance-group
```

若程式支援其他 flag 或環境變數，建議查看 `main.go` 或開發者註解以取得完整參數列表。

---

## Terraform 使用 🌱
Terraform 配置位於 `terraform/` 目錄。

基本流程：

```bash
cd terraform
terraform init
terraform plan -var-file=terraform.tfvars
terraform apply -var-file=terraform.tfvars
```

- 若使用遠端 state 或 GCS backend，請先確認 `provider.tf` 與其他 backend 設定。
- `terraform.tfvars` 可用來置放專案 ID、地區、機器規格等設定。

---

## 設定說明 ⚙️
- `terraform/terraform.tfvars`：放置專案特定變數（例如 `project_id`, `region` 等）。
- 若需要 secret，可以使用環境變數或 GCP Secret Manager 並在 Terraform/程式中使用。

---

## 測試與除錯 🐞
- 建議先在 sandbox 專案測試 Terraform 變更。
- 查看程式 log（如有）以獲得更多執行細節。

---

## 貢獻 🤝
歡迎透過 Issue 或 Pull Request 貢獻：
1. Fork 專案
2. 建立 feature branch
3. 發送 PR 並描述變更

---

## 授權 📄
本專案採用 **MIT 授權**。如需替換授權或加入作者資訊，請更新 `LICENSE` 檔案中的版權宣告（預設為 `Copyright (c) 2026 <wadeliu>`）。

---

如需我把 README 改成英文版、加入範例 output、或針對 `main.go` 裡的 flag/環境變數補充具體使用方式，請告訴我要補充的細節。✅
