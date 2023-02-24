data "google_compute_instance_group_manager" "instance-group" {
  name = var.group_name
  zone = "asia-east1-b"
}

resource "google_compute_autoscaler" "autoscaler" {
  provider = google-beta

  name   = data.google_compute_instance_group_manager.instance-group.name
  zone   = data.google_compute_instance_group_manager.instance-group.zone
  target = data.google_compute_instance_group_manager.instance-group.id

  autoscaling_policy {
    min_replicas    = 1
    max_replicas    = var.max_replicas
    cooldown_period = 600

    metric {
      name   = "custom.googleapis.com/opencensus/process-exists"
      target = 0.001
      type   = "DELTA_PER_SECOND"
      filter = "metric.labels.group_name = \"${var.group_name}\""
    }
  }
}
