# gar-credential-provider

A kubelet credential provider plugin for Google Artifact Registry that enables Kubernetes nodes (EKS, AKS, or any kubelet) to pull container images from GCP Artifact Registry using Workload Identity Federation — no static credentials required.

## How it works

```
kubelet (image pull request for *.pkg.dev)
    │
    ▼
gar-credential-provider (this binary)
    │
    ├─► Read GCP credential config (WIF settings)
    ├─► Get cloud credentials (AWS IMDS)
    ├─► Exchange for GCP access token via STS
    │
    ▼
Return Docker credentials to kubelet
```

## Prerequisites

### GCP Setup

1. **Workload Identity Pool** with AWS provider configured
2. **Service Account** with `roles/artifactregistry.reader` role
3. **WIF binding** allowing your AWS account to impersonate the service account

```hcl
# Service account for pulling images
resource "google_service_account" "gar_reader" {
  account_id   = "eks-gar-reader"
  display_name = "EKS GAR Reader"
}

# Grant artifact registry reader role
resource "google_project_iam_member" "ar_reader" {
  project = var.project_id
  role    = "roles/artifactregistry.reader"
  member  = "serviceAccount:${google_service_account.gar_reader.email}"
}

# Allow AWS principals to impersonate this SA
resource "google_service_account_iam_member" "wif_binding" {
  service_account_id = google_service_account.gar_reader.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/projects/${var.project_number}/locations/global/workloadIdentityPools/${var.wif_pool_id}/attribute.account/${var.aws_account_id}"
}
```

## Installation

Install via Helm chart. The chart deploys a DaemonSet that automatically configures each node.

```bash
helm repo add mondu https://mondu-ai.github.io/helm-charts-community
helm repo update

helm install gar-credential-provider mondu/gar-credential-provider \
  --namespace kube-system \
  --set gcp.audience="//iam.googleapis.com/projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/POOL_ID/providers/PROVIDER_ID" \
  --set gcp.serviceAccountEmail="gar-reader@your-project.iam.gserviceaccount.com"
```

The DaemonSet will:
1. Run once on each node (including new nodes)
2. Install the binary and configuration
3. Restart kubelet
4. Label the node and exit (no running pods left)

**Upgrading:** Update the image tag and the DaemonSet will automatically reconfigure nodes with the old version.

## Docker Image

Multi-arch images (linux/amd64, linux/arm64):
```
ghcr.io/mondu-ai/gar-credential-provider:latest
ghcr.io/mondu-ai/gar-credential-provider:<version>
```

## Troubleshooting

### Check if credential provider is installed

```bash
# On the node
ls -la /etc/eks/image-credential-provider/
cat /etc/eks/image-credential-provider/config.json
```

### Test credential provider manually

```bash
echo '{"apiVersion":"credentialprovider.kubelet.k8s.io/v1","kind":"CredentialProviderRequest","image":"europe-west3-docker.pkg.dev/myproject/repo/image:tag"}' | \
  /etc/eks/image-credential-provider/gar-credential-provider --config=/etc/eks/image-credential-provider/gcp-credential-config.json
```

### Check kubelet logs

```bash
journalctl -u kubelet | grep -i credential
```

### Common errors

| Error | Cause | Solution |
|-------|-------|----------|
| "failed to get token" | WIF misconfigured | Check audience and provider settings |
| "permission denied" | SA missing role | Grant `roles/artifactregistry.reader` |

## Development

```bash
make build           # Build for current platform
make build-all       # Build for linux/amd64 and linux/arm64
make test            # Run tests
make lint            # Run linter
```

## Release

Releases are created automatically when a tag is pushed:

```bash
git tag v1.0.0
git push origin v1.0.0
```

This will:
- Build binaries for linux/amd64 and linux/arm64 (GitHub Releases)
- Build and push Docker images to `ghcr.io/mondu-ai/gar-credential-provider`

## License

MIT
