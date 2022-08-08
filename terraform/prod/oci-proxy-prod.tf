/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

locals {
  project_id = "k8s-infra-oci-proxy-prod"
  domain     = "registry.k8s.io"
  image      = "gcr.io/k8s-staging-infra-tools/archeio:${var.tag}"

}

data "google_billing_account" "account" {
  billing_account = "018801-93540E-22A20E"
}

data "google_organization" "org" {
  domain = "kubernetes.io"
}

resource "google_project" "project" {
  name            = local.project_id
  project_id      = local.project_id
  org_id          = data.google_organization.org.org_id
  billing_account = data.google_billing_account.account.id
}

// Enable services needed for the project
resource "google_project_service" "project" {
  project = google_project.project.id

  for_each = toset([
    "compute.googleapis.com",
    "containerregistry.googleapis.com",
    "logging.googleapis.com",
    "oslogin.googleapis.com",
    "pubsub.googleapis.com",
    "run.googleapis.com",
    "storage-api.googleapis.com",
    "storage-component.googleapis.com"
  ])

  service = each.key
}

// Ensure k8s-infra-oci-proxy-admins@kubernetes.io has admin access to this project
resource "google_project_iam_member" "k8s_infra_oci_proxy_admins" {
  project = google_project.project.id
  role    = "roles/owner"
  member  = "group:k8s-infra-oci-proxy-admins@kubernetes.io"
}

resource "google_service_account" "oci-proxy" {
  project      = google_project.project.project_id
  account_id   = "oci-proxy-prod"
  display_name = "Minimal Service Account for OCI Proxy"
}

resource "google_cloud_run_service_iam_member" "allUsers" {
  project  = google_project.project.project_id
  for_each = google_cloud_run_service.oci-proxy

  service  = google_cloud_run_service.oci-proxy[each.key].name
  location = google_cloud_run_service.oci-proxy[each.key].location
  role     = "roles/run.invoker"
  member   = "allUsers"

  depends_on = [google_cloud_run_service.oci-proxy]
}

resource "google_cloud_run_service" "oci-proxy" {
  project  = google_project.project.project_id
  for_each = toset(var.cloud_run_regions)
  name     = "${local.project_id}-${each.key}"
  location = each.key

  template {
    metadata {
      annotations = {
        "autoscaling.knative.dev/maxScale" = "5" // Control costs.
        "run.googleapis.com/launch-stage"  = "BETA"
      }
    }
    spec {
      service_account_name = google_service_account.oci-proxy.email
      containers {
        image = local.image
      }
      container_concurrency = 10
      // 30 seconds less than cloud scheduler maximum.
      timeout_seconds = 570
    }
  }

  traffic {
    percent         = 100
    latest_revision = true
  }

  depends_on = [
    google_project_service.project["run.googleapis.com"]
  ]

  lifecycle {
    ignore_changes = [
      // This gets added by the Cloud Run API post deploy and causes diffs, can be ignored...
      template[0].metadata[0].annotations["run.googleapis.com/sandbox"],
    ]
  }
}
