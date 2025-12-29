# Terraform Guide for Hauslet Services

Complete guide to using Terraform to manage Hauslet infrastructure.

## Table of Contents

- [What is Terraform?](#what-is-terraform)
- [Project Structure](#project-structure)
- [Basic Concepts](#basic-concepts)
- [Common Workflows](#common-workflows)
- [Managing State](#managing-state)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## What is Terraform?

**Terraform** is an Infrastructure as Code (IaC) tool that lets you define and manage cloud infrastructure using declarative configuration files.

### Why Use Terraform?

**Without Terraform (Manual):**
```bash
# Error-prone, hard to track, no history
gcloud sql instances create db1 ...
gcloud run services create api1 ...
gcloud scheduler jobs create job1 ...
# What did I create? How do I recreate this? 🤷
```

**With Terraform (Declarative):**
```hcl
# infrastructure.tf - Everything documented
resource "google_sql_database_instance" "db" {
  name = "hauslet-db"
  # ...
}

resource "google_cloud_run_service" "api" {
  name = "hauslet-api"
  # ...
}
```

```bash
terraform apply  # Creates everything
# ✅ Documented, versioned, reproducible
```

### Benefits

| Benefit | Description |
|---------|-------------|
| **Version Control** | Track infrastructure changes in Git |
| **Reproducible** | Deploy identical environments (dev/staging/prod) |
| **Collaborative** | Team reviews infrastructure changes via PRs |
| **State Tracking** | Terraform knows what exists and what changed |
| **Plan Before Apply** | Preview changes before making them |
| **Automation** | Integrate with CI/CD pipelines |

## Project Structure

```
deploy/terraform/
├── provider.tf           # Terraform & provider configuration
├── variables.tf          # Input variable definitions
├── terraform.tfvars      # Actual values (secrets, project IDs)
├── cloudscheduler.tf     # Cloud Scheduler jobs
├── cicd.tf               # CI/CD infrastructure (service accounts, etc.)
├── outputs.tf            # Values to display after apply
├── .terraform/           # Provider plugins (auto-generated)
├── .terraform.lock.hcl   # Provider version lock
└── terraform.tfstate     # Current infrastructure state (CRITICAL!)
```

### File Purposes

**provider.tf** - Configure Terraform and providers:
```hcl
terraform {
  required_version = ">= 1.5"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 7.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}
```

**variables.tf** - Define input variables:
```hcl
variable "project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "region" {
  description = "GCP region"
  type        = string
  default     = "europe-north1"
}
```

**terraform.tfvars** - Set actual values:
```hcl
project_id = "gen-lang-client-0265949535"
region     = "europe-north1"
worker_url = "https://hauslet-worker-xxx.run.app"
```

**cloudscheduler.tf** - Define resources:
```hcl
resource "google_cloud_scheduler_job" "cleanup" {
  name     = "cleanup-job"
  schedule = "*/15 * * * *"
  # ...
}
```

**outputs.tf** - Display important info:
```hcl
output "api_url" {
  value = google_cloud_run_service.api.status[0].url
}
```

## Basic Concepts

### Resources

Resources are infrastructure components you want to create/manage.

```hcl
resource "PROVIDER_TYPE" "NAME" {
  # Configuration
}

# Example:
resource "google_cloud_scheduler_job" "media_cleanup" {
  name        = "media-cleanup-scheduler"
  description = "Clean up old media files"
  schedule    = "*/15 * * * *"
  region      = "europe-west1"

  http_target {
    uri         = "https://worker.example.com/cleanup"
    http_method = "POST"
  }
}
```

### Variables

Variables make your configuration reusable.

```hcl
# Define
variable "environment" {
  type    = string
  default = "staging"
}

# Use
resource "google_cloud_run_service" "api" {
  name = "hauslet-api-${var.environment}"
}
```

### Data Sources

Read information from existing resources.

```hcl
# Get current project info
data "google_project" "current" {}

# Use project number
resource "some_resource" "example" {
  project_number = data.google_project.current.number
}
```

### Outputs

Display values after deployment.

```hcl
output "scheduler_jobs" {
  value = {
    cleanup = google_cloud_scheduler_job.media_cleanup.name
    expiry  = google_cloud_scheduler_job.booking_expiry.name
  }
}
```

### Dependencies

Terraform automatically detects dependencies:

```hcl
# Implicit dependency (via reference)
resource "google_cloud_scheduler_job" "job" {
  # References var.worker_url - Terraform knows job depends on worker
  http_target {
    uri = var.worker_url
  }
}

# Explicit dependency
resource "google_cloud_scheduler_job" "job" {
  # ...
  depends_on = [
    google_project_service.cloudscheduler  # Must enable API first
  ]
}
```

## Common Workflows

### Initial Setup

```bash
# 1. Navigate to terraform directory
cd deploy/terraform

# 2. Initialize Terraform (download providers)
terraform init

# Output:
# Initializing the backend...
# Initializing provider plugins...
# - Installing hashicorp/google v7.14.1...
# Terraform has been successfully initialized!

# 3. Validate configuration
terraform validate

# Output:
# Success! The configuration is valid.
```

### Making Changes

```bash
# 1. Edit configuration files
vim cloudscheduler.tf

# 2. Format code (optional but recommended)
terraform fmt

# 3. Validate syntax
terraform validate

# 4. Plan changes (preview what will happen)
terraform plan

# Output shows:
# + create    (resource will be created)
# - destroy   (resource will be destroyed)
# ~ update    (resource will be modified)
# -/+ replace (resource will be destroyed and recreated)

# 5. Apply changes
terraform apply

# Review plan and type 'yes' to confirm
# Or auto-approve:
terraform apply -auto-approve
```

### Viewing Current State

```bash
# List all resources
terraform state list

# Show specific resource
terraform state show google_cloud_scheduler_job.media_cleanup

# Show all outputs
terraform output

# Show specific output
terraform output scheduler_jobs
```

### Importing Existing Resources

If you created resources manually, import them to Terraform:

```bash
# 1. Add resource to .tf file (without values)
resource "google_cloud_scheduler_job" "existing_job" {
  # Will be populated from import
}

# 2. Import the resource
terraform import google_cloud_scheduler_job.existing_job \
  projects/PROJECT_ID/locations/REGION/jobs/JOB_NAME

# 3. Run plan to see current state
terraform plan

# 4. Update .tf file to match actual resource
# (Use values from plan output)

# 5. Verify no changes needed
terraform plan
# Should show: No changes. Infrastructure is up-to-date.
```

### Destroying Resources

```bash
# Destroy specific resource
terraform destroy -target=google_cloud_scheduler_job.media_cleanup

# Destroy everything (CAREFUL!)
terraform destroy

# Preview what will be destroyed
terraform plan -destroy
```

## Managing State

### What is State?

`terraform.tfstate` is a JSON file tracking:
- What resources exist
- Their current configuration
- Dependencies between resources

**CRITICAL:** Never edit `terraform.tfstate` manually!

### Local State (Current Setup)

```bash
# State stored in: deploy/terraform/terraform.tfstate
# ⚠️  Add to .gitignore (contains sensitive info)
# ⚠️  Backup regularly
```

### Remote State (Recommended for Teams)

Store state in Google Cloud Storage for collaboration:

```hcl
# In provider.tf
terraform {
  backend "gcs" {
    bucket = "hauslet-terraform-state"
    prefix = "terraform/state"
  }
}
```

Setup:
```bash
# Create state bucket
gsutil mb -l europe-north1 gs://hauslet-terraform-state

# Enable versioning (for rollback)
gsutil versioning set on gs://hauslet-terraform-state

# Migrate local state to remote
terraform init -migrate-state
```

### State Commands

```bash
# List resources in state
terraform state list

# Show resource details
terraform state show google_cloud_scheduler_job.media_cleanup

# Remove resource from state (doesn't delete actual resource)
terraform state rm google_cloud_scheduler_job.media_cleanup

# Move resource to different name
terraform state mv \
  google_cloud_scheduler_job.old_name \
  google_cloud_scheduler_job.new_name

# Pull remote state
terraform state pull > terraform.tfstate.backup

# Push local state to remote
terraform state push terraform.tfstate
```

## Best Practices

### 1. Always Plan Before Apply

```bash
# GOOD
terraform plan
terraform apply

# BAD
terraform apply -auto-approve  # Only use in CI/CD
```

### 2. Use Variables for Reusability

```hcl
# BAD - Hardcoded
resource "google_cloud_run_service" "api" {
  name = "hauslet-api-staging"
}

# GOOD - Parameterized
resource "google_cloud_run_service" "api" {
  name = "hauslet-api-${var.environment}"
}
```

### 3. Keep State Secure

```bash
# .gitignore
terraform.tfstate
terraform.tfstate.backup
*.tfvars  # Contains secrets
.terraform/
```

### 4. Use Meaningful Names

```hcl
# BAD
resource "google_cloud_scheduler_job" "job1" { }

# GOOD
resource "google_cloud_scheduler_job" "media_cleanup" { }
```

### 5. Document Complex Resources

```hcl
# Cloud Scheduler job to clean up media older than 2 hours
# Runs every 15 minutes to prevent storage bloat
# Related handler: internal/transport/worker/handlers/listing/media_cleanup.go
resource "google_cloud_scheduler_job" "media_cleanup" {
  # ...
}
```

### 6. Use Workspaces for Environments

```bash
# Create workspaces
terraform workspace new staging
terraform workspace new production

# Switch workspace
terraform workspace select staging

# Each workspace has separate state
```

### 7. Version Your Providers

```hcl
terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 7.0"  # Allow 7.x updates, not 8.x
    }
  }
}
```

### 8. Use Modules for Reusability

```hcl
# modules/cloud-run-service/main.tf
variable "service_name" {}
variable "image" {}

resource "google_cloud_run_service" "service" {
  name = var.service_name
  # ...
}

# Use module
module "api_service" {
  source       = "./modules/cloud-run-service"
  service_name = "hauslet-api"
  image        = "gcr.io/project/api:latest"
}
```

## Troubleshooting

### Error: Resource Already Exists

```
Error 409: Service account already exists
```

**Solution:** Import existing resource:
```bash
terraform import google_service_account.api \
  projects/PROJECT_ID/serviceAccounts/EMAIL
```

### Error: Provider Not Found

```
Error: Could not load plugin
```

**Solution:** Re-initialize:
```bash
rm -rf .terraform
terraform init
```

### Error: State Lock

```
Error: Error acquiring state lock
```

**Solution:** Force unlock (if you're sure no one else is running):
```bash
terraform force-unlock LOCK_ID
```

### Error: Invalid Provider Configuration

```
Error: Attempted to load application default credentials
```

**Solution:** Authenticate:
```bash
gcloud auth application-default login
```

### Drift Detection

Check if resources changed outside Terraform:

```bash
terraform plan -refresh-only

# If drift detected, update state:
terraform apply -refresh-only
```

### Debugging

```bash
# Enable detailed logging
export TF_LOG=DEBUG
terraform plan

# Save log to file
export TF_LOG_PATH=terraform.log
terraform plan

# Disable logging
unset TF_LOG
```

## Advanced Features

### Terraform Functions

```hcl
# String manipulation
name = upper("hauslet")  # "HAUSLET"
name = lower("HAUSLET")  # "hauslet"

# Lists
names = concat(["api"], ["worker"])  # ["api", "worker"]

# Conditionals
environment = var.env == "prod" ? "production" : "staging"

# Encoding
body = base64encode(jsonencode({ key = "value" }))
```

### For Each

```hcl
variable "scheduler_jobs" {
  type = map(object({
    schedule = string
    endpoint = string
  }))
  default = {
    cleanup = {
      schedule = "*/15 * * * *"
      endpoint = "/tasks/media/cleanup"
    }
    expiry = {
      schedule = "*/2 * * * *"
      endpoint = "/tasks/booking/expiry"
    }
  }
}

resource "google_cloud_scheduler_job" "jobs" {
  for_each = var.scheduler_jobs

  name     = "${each.key}-scheduler"
  schedule = each.value.schedule

  http_target {
    uri = "${var.worker_url}${each.value.endpoint}"
  }
}
```

### Dynamic Blocks

```hcl
resource "google_cloud_run_service" "api" {
  name = "api"

  # Create multiple env vars
  dynamic "template" {
    for_each = var.env_vars
    content {
      env {
        name  = template.key
        value = template.value
      }
    }
  }
}
```

### Provisioners (Use Sparingly)

```hcl
resource "google_cloud_run_service" "api" {
  # ...

  # Run command after creation
  provisioner "local-exec" {
    command = "echo Service URL: ${self.status[0].url}"
  }
}
```

## Useful Commands Reference

```bash
# Initialization
terraform init          # Initialize working directory
terraform init -upgrade # Upgrade providers

# Planning
terraform plan                    # Preview changes
terraform plan -out=plan.tfplan   # Save plan to file
terraform apply plan.tfplan       # Apply saved plan
terraform plan -destroy           # Preview destroy

# Applying
terraform apply               # Apply changes (requires confirmation)
terraform apply -auto-approve # Apply without confirmation
terraform apply -target=RESOURCE # Apply specific resource

# State
terraform state list              # List resources
terraform state show RESOURCE     # Show resource details
terraform state rm RESOURCE       # Remove from state
terraform state mv SRC DEST       # Rename resource
terraform state pull              # Download remote state
terraform state push              # Upload state

# Outputs
terraform output              # Show all outputs
terraform output OUTPUTNAME   # Show specific output

# Workspaces
terraform workspace list      # List workspaces
terraform workspace new NAME  # Create workspace
terraform workspace select NAME # Switch workspace
terraform workspace delete NAME # Delete workspace

# Validation
terraform validate # Validate configuration
terraform fmt      # Format code
terraform fmt -check # Check if formatted

# Destroy
terraform destroy              # Destroy all resources
terraform destroy -target=RES  # Destroy specific resource

# Import
terraform import ADDRESS ID    # Import existing resource

# Other
terraform show           # Show current state
terraform graph          # Generate dependency graph
terraform version        # Show Terraform version
terraform providers      # Show required providers
```

## Resources

- [Terraform Documentation](https://www.terraform.io/docs)
- [Google Provider Docs](https://registry.terraform.io/providers/hashicorp/google/latest/docs)
- [Terraform Best Practices](https://www.terraform-best-practices.com/)
- [HCL Syntax](https://www.terraform.io/docs/language/syntax/configuration.html)
