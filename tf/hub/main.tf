/* Variables */

variable "project_id" {
  description = "Project id (also used for the Apigee Organization)."
  type        = string
}

variable "region" {
  description = "GCP region for the Apigee runtime & analytics data."
  type        = string
}

/* API Hub */

resource "google_apihub_host_project_registration" "apihub_host_project" {
  project                      = var.project_id
  location                     = var.region
  host_project_registration_id = var.project_id
  gcp_project                  = "projects/${var.project_id}"
}

resource "google_project_service_identity" "apihub_service_identity" {
  provider = google-beta
  project  = var.project_id
  service  = "apihub.googleapis.com"
}

resource "google_project_iam_member" "apihub_service_identity_permission" {
  provider = google-beta
  project  = var.project_id
  for_each = toset([
    "roles/apihub.admin",
    "roles/apihub.runtimeProjectServiceAgent"
  ])
  role       = each.key
  member     = "serviceAccount:${google_project_service_identity.apihub_service_identity.email}"
  depends_on = [google_project_service_identity.apihub_service_identity]
}

resource "google_apihub_api_hub_instance" "apihub-instance" {
  provider = google-beta
  project  = var.project_id
  location = var.region
  config {
    disable_search = true
  }
  depends_on = [google_apihub_host_project_registration.apihub_host_project]
}
