# GAR Credential Provider

[![Go Version](https://img.shields.io/github/go-mod/go-version/mondu-ai/gar-credential-provider)](https://golang.org/)
[![License](https://img.shields.io/github/license/mondu-ai/gar-credential-provider)](LICENSE)
[![Container](https://img.shields.io/badge/container-ghcr.io-blue)](https://github.com/mondu-ai/gar-credential-provider/pkgs/container/gar-credential-provider)
[![Go Report Card](https://goreportcard.com/badge/github.com/mondu-ai/gar-credential-provider)](https://goreportcard.com/report/github.com/mondu-ai/gar-credential-provider)

Pull images from Google Artifact Registry on EKS clusters using Workload Identity Federation.

## Why?

You've got an EKS cluster, but your container images live in Google Artifact Registry. The cluster can't pull them because it doesn't have GCP credentials.

The old way: create a GCP service account key, store it as a Kubernetes secret, and configure `imagePullSecrets` on every deployment. That's a lot of YAML, and now you've got a credential that never expires sitting in your cluster.

This tool takes a different approach. It uses [Workload Identity Federation](https://cloud.google.com/iam/docs/workload-identity-federation) to exchange the node's AWS IAM role for a short-lived GCP access token. No long-lived keys. No secrets to rotate. No `imagePullSecrets` everywhere.

## How it works

```
kubelet needs image from *.pkg.dev
         │
         ▼
┌─────────────────────────────────┐
│   gar-credential-provider       │
│                                 │
│  1. Read AWS credentials        │
│     (from IMDS)                 │
│                                 │
│  2. Exchange for GCP token      │
│     (via STS)                   │
│                                 │
│  3. Return Docker credentials   │
│     (oauth2accesstoken:token)   │
└─────────────────────────────────┘
         │
         ▼
   kubelet pulls image
```

The binary implements the [kubelet credential provider API](https://kubernetes.io/docs/reference/config-api/kubelet-credentialprovider.v1/). When kubelet tries to pull an image from a matching registry (like `us-central1-docker.pkg.dev`), it calls this binary via stdin/stdout. The binary fetches a GCP access token and returns it as Docker credentials.

Tokens are cached by kubelet (default: 50 minutes), so you're not hitting the STS endpoint on every pull.

## Supported credential sources

| Source | How it works |
|--------|--------------|
| **AWS (EKS)** | Reads credentials from IMDS, signs GetCallerIdentity, GCP verifies with AWS STS |
| **OIDC** | Reads JWT from file or URL, GCP verifies signature (for other clusters) |

## Installation

### Helm (recommended)

```bash
helm repo add mondu https://mondu-ai.github.io/helm-charts-community
helm repo update

helm install gar-credential-provider mondu/gar-credential-provider \
  --namespace kube-system \
  --set gcp.audience="//iam.googleapis.com/projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/POOL_ID/providers/PROVIDER_ID" \
  --set gcp.serviceAccountEmail="gar-reader@your-project.iam.gserviceaccount.com"
```

That's it. The chart deploys a DaemonSet that configures each node automatically.

### What the DaemonSet does

Under the hood, the binary runs in `install` mode on each node:

1. Copies itself to `/etc/eks/image-credential-provider/`
2. Writes the GCP WIF credential config
3. Updates kubelet's credential provider configuration
4. Restarts kubelet to pick up changes
5. Labels the node with `gar-credential-provider/version`

New nodes get configured automatically when they join the cluster.

## GCP Workload Identity Federation setup

### 1. Create a Workload Identity Pool

```bash
gcloud iam workload-identity-pools create my-pool \
  --location="global" \
  --display-name="My Pool"
```

### 2. Create an AWS provider

```bash
gcloud iam workload-identity-pools providers create-aws aws-provider \
  --location="global" \
  --workload-identity-pool="my-pool" \
  --account-id="AWS_ACCOUNT_ID"
```

### 3. Grant Artifact Registry access

```bash
gcloud artifacts repositories add-iam-policy-binding my-repo \
  --location="us-central1" \
  --member="principalSet://iam.googleapis.com/projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/my-pool/attribute.aws_role/arn:aws:sts::AWS_ACCOUNT_ID:assumed-role/my-eks-node-role" \
  --role="roles/artifactregistry.reader"
```

### 4. Download the credential config

```bash
gcloud iam workload-identity-pools create-cred-config \
  projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/my-pool/providers/aws-provider \
  --aws \
  --output-file=gcp-credential-config.json
```

## Manual installation

If you'd rather not use Helm, grab the binary from [releases](https://github.com/mondu-ai/gar-credential-provider/releases) and drop it in kubelet's credential provider directory.

Create the GCP WIF config file (see config examples below).

Update kubelet's `--image-credential-provider-config`:

```json
{
  "apiVersion": "kubelet.config.k8s.io/v1",
  "kind": "CredentialProviderConfig",
  "providers": [
    {
      "name": "gar-credential-provider",
      "matchImages": ["*.pkg.dev"],
      "defaultCacheDuration": "50m",
      "apiVersion": "credentialprovider.kubelet.k8s.io/v1",
      "args": ["--config=/etc/eks/image-credential-provider/gcp-credential-config.json"]
    }
  ]
}
```

Restart kubelet.

### Config examples

<details>
<summary>AWS (EKS)</summary>

```json
{
  "type": "external_account",
  "audience": "//iam.googleapis.com/projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/POOL_ID/providers/aws-PROVIDER_ID",
  "subject_token_type": "urn:ietf:params:aws:token-type:aws4_request",
  "token_url": "https://sts.googleapis.com/v1/token",
  "credential_source": {
    "environment_id": "aws1",
    "regional_cred_verification_url": "https://sts.{region}.amazonaws.com?Action=GetCallerIdentity&Version=2011-06-15"
  }
}
```
</details>

<details>
<summary>OIDC token file</summary>

```json
{
  "type": "external_account",
  "audience": "//iam.googleapis.com/projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/POOL_ID/providers/PROVIDER_ID",
  "subject_token_type": "urn:ietf:params:oauth:token-type:jwt",
  "token_url": "https://sts.googleapis.com/v1/token",
  "credential_source": {
    "file": "/var/run/secrets/tokens/gcp-token"
  }
}
```
</details>

## License

MIT
