# Hauslet Services Documentation Index

Welcome to the Hauslet Services deployment and infrastructure documentation.

## 📚 Documentation Overview

This directory contains comprehensive guides for deploying, managing, and maintaining Hauslet Services on Google Cloud Platform.

## 🗂️ Quick Navigation

### Getting Started

| Document | Description | When to Read |
|----------|-------------|--------------|
| [Deployment Guide](./DEPLOYMENT_GUIDE.md) | Complete deployment walkthrough | First time deploying |
| [README](./README.md) | Project overview and quick start | Project introduction |

### Infrastructure as Code

| Document | Description | When to Read |
|----------|-------------|--------------|
| [Terraform Guide](./TERRAFORM_GUIDE.md) | Learn Terraform basics and workflows | Before using Terraform |
| [Scheduler Guide](./SCHEDULER_GUIDE.md) | Manage Cloud Scheduler jobs | Adding/modifying cron jobs |

### Reference & Cheatsheets

| Document | Description | When to Read |
|----------|-------------|--------------|
| [gcloud Cheatsheet](./GCLOUD_CHEATSHEET.md) | Quick reference for gcloud commands | Daily operations |
| [CI/CD Guide](./CICD_GUIDE.md) | Automated deployment setup | Setting up automation |

### Planning & Architecture

| Document | Description | When to Read |
|----------|-------------|--------------|
| [GCP Deployment Plan](./GCP_DEPLOYMENT_PLAN.md) | Infrastructure architecture overview | Understanding the system |
| [Deployment Steps](./DEPLOYMENT_STEPS.md) | Step-by-step deployment checklist | Following deployment process |

## 🚀 Quick Start Paths

### Path 1: I Want to Deploy for the First Time

```
1. Read: GCP Deployment Plan (architecture overview)
2. Read: Deployment Guide (complete walkthrough)
3. Use: gcloud Cheatsheet (as reference during setup)
4. Use: Terraform Guide (when editing infrastructure)
```

### Path 2: I Want to Make Code Changes

```
1. Make changes to code
2. Run: ./deploy/deploy-staging.sh
3. Verify deployment in staging
4. Run: ./deploy/deploy-production.sh (if needed)
5. Reference: gcloud Cheatsheet (for troubleshooting)
```

### Path 3: I Want to Add a Scheduler Job

```
1. Read: Scheduler Guide (complete walkthrough)
2. Edit: deploy/terraform/cloudscheduler.tf
3. Run: terraform apply
4. Test: gcloud scheduler jobs run JOB_NAME
5. Reference: gcloud Cheatsheet (for monitoring)
```

### Path 4: I Want to Modify Infrastructure

```
1. Read: Terraform Guide (understand Terraform basics)
2. Edit: deploy/terraform/*.tf files
3. Run: terraform plan (preview changes)
4. Run: terraform apply (apply changes)
5. Reference: gcloud Cheatsheet (verify resources)
```

### Path 5: I Want to Set Up CI/CD

```
1. Read: CI/CD Guide (automation setup)
2. Configure: GitHub triggers
3. Test: Push to staging branch
4. Monitor: Cloud Build console
```

## 📖 Document Details

### [Deployment Guide](./DEPLOYMENT_GUIDE.md)

**What it covers:**
- Complete deployment walkthrough
- Initial GCP setup
- Database configuration
- Secret management
- Staging and production deployment
- Troubleshooting common issues

**Key sections:**
- Prerequisites
- Initial Setup (database, secrets, VPC)
- Staging Deployment
- Production Deployment
- Troubleshooting

**Best for:** First-time deployers, setting up new environments

---

### [Terraform Guide](./TERRAFORM_GUIDE.md)

**What it covers:**
- What Terraform is and why we use it
- Project structure explained
- Basic concepts (resources, variables, state)
- Common workflows
- Best practices
- Troubleshooting

**Key sections:**
- What is Terraform?
- Project Structure
- Common Workflows
- Managing State
- Best Practices

**Best for:** Learning Infrastructure as Code, modifying infrastructure

---

### [gcloud Cheatsheet](./GCLOUD_CHEATSHEET.md)

**What it covers:**
- Quick reference for all gcloud commands
- Cloud Run operations
- Cloud Build commands
- Secret Manager
- IAM and service accounts
- Logging and monitoring

**Key sections:**
- Authentication & Configuration
- Cloud Run
- Cloud Build
- Cloud Scheduler
- Secret Manager
- IAM & Service Accounts
- Logging

**Best for:** Daily operations, quick command lookup

---

### [Scheduler Guide](./SCHEDULER_GUIDE.md)

**What it covers:**
- Understanding Cloud Scheduler
- Current jobs explained
- Adding new jobs step-by-step
- Managing and monitoring jobs
- Cron patterns reference
- Troubleshooting

**Key sections:**
- Current Jobs
- Adding New Jobs
- Managing Jobs
- Monitoring
- Cron Patterns
- Best Practices

**Best for:** Working with background jobs, adding scheduled tasks

---

### [CI/CD Guide](./CICD_GUIDE.md)

**What it covers:**
- Automated deployment setup
- GitHub integration
- Build triggers
- Deployment pipeline

**Best for:** Setting up automation, understanding CI/CD flow

---

### [GCP Deployment Plan](./GCP_DEPLOYMENT_PLAN.md)

**What it covers:**
- Overall architecture
- Component descriptions
- Infrastructure requirements
- Deployment phases

**Best for:** Understanding system architecture, planning deployments

---

## 🔧 Common Tasks

### Deploy to Staging

```bash
cd deploy
./deploy-staging.sh
```

**Reference:** [Deployment Guide § Staging Deployment](./DEPLOYMENT_GUIDE.md#staging-deployment)

### View Service Logs

```bash
gcloud run services logs read hauslet-api-staging \
  --region=europe-north1 \
  --limit=50
```

**Reference:** [gcloud Cheatsheet § Cloud Run Logs](./GCLOUD_CHEATSHEET.md#logs)

### Add New Scheduler Job

```bash
# 1. Edit cloudscheduler.tf
vim deploy/terraform/cloudscheduler.tf

# 2. Apply changes
cd deploy/terraform
terraform apply
```

**Reference:** [Scheduler Guide § Adding New Jobs](./SCHEDULER_GUIDE.md#adding-new-jobs)

### Update Infrastructure

```bash
cd deploy/terraform

# Review changes
terraform plan

# Apply changes
terraform apply
```

**Reference:** [Terraform Guide § Making Changes](./TERRAFORM_GUIDE.md#making-changes)

### Check Job Status

```bash
gcloud scheduler jobs list --location=europe-west1
```

**Reference:** [Scheduler Guide § Managing Jobs](./SCHEDULER_GUIDE.md#managing-jobs)

### Rotate Secrets

```bash
echo -n "new-secret-value" | \
  gcloud secrets versions add SECRET_NAME --data-file=-
```

**Reference:** [gcloud Cheatsheet § Secret Manager](./GCLOUD_CHEATSHEET.md#secret-manager)

## 🆘 Getting Help

### When Things Go Wrong

1. **Check logs first:**
   ```bash
   gcloud run services logs read SERVICE_NAME --limit=50
   ```

2. **Common issues:**
   - Service won't start → [Deployment Guide § Troubleshooting](./DEPLOYMENT_GUIDE.md#troubleshooting)
   - Scheduler not running → [Scheduler Guide § Troubleshooting](./SCHEDULER_GUIDE.md#troubleshooting)
   - Terraform errors → [Terraform Guide § Troubleshooting](./TERRAFORM_GUIDE.md#troubleshooting)

3. **Still stuck?**
   - Check GCP Console for detailed error messages
   - Review recent changes in Git history
   - Contact DevOps team

### Quick Diagnostics

```bash
# Check service status
gcloud run services list --region=europe-north1

# Check recent errors
gcloud logging read "severity>=ERROR" --limit=20

# Check scheduler jobs
gcloud scheduler jobs list --location=europe-west1

# Check Cloud Build status
gcloud builds list --region=europe-north1 --limit=5
```

## 📝 Contributing to Docs

When updating documentation:

1. **Keep it practical** - Focus on how-to, not theory
2. **Include examples** - Show real commands, not just syntax
3. **Update index** - Keep this file current
4. **Test commands** - Verify commands actually work
5. **Add context** - Explain why, not just what

### Documentation Standards

- Use markdown formatting
- Include code blocks with language hints
- Add tables for comparisons
- Use emojis sparingly for visual navigation
- Keep line length reasonable (80-100 chars)

## 🔗 External Resources

### Google Cloud

- [GCP Console](https://console.cloud.google.com)
- [Cloud Run Documentation](https://cloud.google.com/run/docs)
- [Cloud Scheduler Documentation](https://cloud.google.com/scheduler/docs)
- [Secret Manager Documentation](https://cloud.google.com/secret-manager/docs)

### Terraform

- [Terraform Documentation](https://www.terraform.io/docs)
- [Google Provider Reference](https://registry.terraform.io/providers/hashicorp/google/latest/docs)
- [Terraform Best Practices](https://www.terraform-best-practices.com/)

### Tools

- [crontab.guru](https://crontab.guru/) - Cron expression editor
- [Terraform Language Server](https://github.com/hashicorp/terraform-ls) - VS Code extension
- [gcloud CLI Reference](https://cloud.google.com/sdk/gcloud/reference)

## 📊 Document Metadata

| Document | Last Updated | Maintainer |
|----------|-------------|------------|
| DEPLOYMENT_GUIDE.md | 2025-12-28 | DevOps Team |
| TERRAFORM_GUIDE.md | 2025-12-28 | DevOps Team |
| GCLOUD_CHEATSHEET.md | 2025-12-28 | DevOps Team |
| SCHEDULER_GUIDE.md | 2025-12-28 | DevOps Team |
| CI/CD_GUIDE.md | Previous session | DevOps Team |

---

**Need to add a new document?**

1. Create the markdown file in `deploy/docs/`
2. Add entry to this index
3. Link from relevant sections
4. Update Quick Navigation table
5. Commit with descriptive message

---

*Last updated: 2025-12-28*
*For questions or improvements, contact the DevOps team*
