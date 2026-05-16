# AI Gateway Labs on Google Cloud
These labs guide you through creating an AI Gateway on Google Cloud leveraging [Apigee](https://cloud.google.com/solutions/apigee-ai), [Model Garden](https://cloud.google.com/model-garden), [Google Cloud Networking](https://docs.cloud.google.com/docs/networking) and [Security Services](https://docs.cloud.google.com/docs/security/overview/whitepaper) to secure & govern AI traffic to and from models, tools and agents. This approach integrates easily into [Gemini Enterprise Agent Platform](https://docs.cloud.google.com/gemini-enterprise-agent-platform) and [Agent Gateway](https://docs.cloud.google.com/gemini-enterprise-agent-platform/govern/gateways/agent-gateway-overview), as well as to any other models, tools or agent platforms, as well as to [multi-cloud or on-premise](https://cloud.google.com/apigee/hybrid) environments.

![AI Gateway Overview](https://iili.io/Bm91xHB.png)

## Google Services used
* [Apigee](https://cloud.google.com/solutions/apigee-ai) - Apigee is used as the gateway, either running natively on Google Cloud, or as a hybrid deployment in Kubernetes.
* [Gemini Enterprise Model Garden](https://cloud.google.com/model-garden) - Highly efficient model hosting, from SOTA frontier models to leading open source and open weight models.
* [Model Armor](https://cloud.google.com/security/products/model-armor) - Google Cloud Model Armor is used to screen prompts and responses for objectionable content with flexible templates.
*  [Sensitive Data Protection](https://cloud.google.com/security/products/sensitive-data-protection) - Sensitive Data Protection is used to mask PII or other senstitive data from requests and responses.
* [Google Cload Networking](https://docs.cloud.google.com/docs/networking) - High performance [Load Balancers](https://docs.cloud.google.com/load-balancing/docs/load-balancing-overview) are used for ingress and TLS termination, as well as [Private Service Connect](https://docs.cloud.google.com/vpc/docs/private-service-connect) for low-latency, internal traffic & model routing.

## Prerequisites
To run these labs you will need:
* A pre-created [Google Cloud sandbox project](https://docs.cloud.google.com/resource-manager/docs/creating-managing-projects) with either the role **roles/owner** or these roles assigned (**apigee.admin, apihub.admin, serviceusage.serviceUsageAdmin, iam.serviceAccountAdmin, compute.admin, compute.networkAdmin, cloudkms.admin, roles/ml.admin**).

## Cloud Shell labs

The labs are organized as [Google Cloud Shell Tutorials](https://docs.cloud.google.com/shell/docs/cloud-shell-tutorials/tutorials), and can run interactively in your own project.

### 🧱 AI Gateway Foundations lab

In this lab you will:
* Provision an **Apigee X** instance (either as Evaluation, Pay-as-you-go, or Subscription) in your chosen Google Cloud project and region using **Terraform**, using either the global or regional DRZ (US, EU, IN) control planes.
* Deploy AI proxy templates in YAML format to models in your project's [Model Garden](https://cloud.google.com/model-garden) (Gemini, Claude, DeepSeek, Qwen, etc...).
* Use the AI Gateway as proxy in [Gemini CLI](https://geminicli.com/) and [Claude Code](https://claude.com/product/claude-code) for local CLI usage.
* Test failover from an unapproved model to an approved, backup model.
* View analytics usage of each model, for example counts of **prompt tokens**, **response tokens**, **time-to-first-token**, **latency**, etc...

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.png)](https://ssh.cloud.google.com/cloudshell/open?cloudshell_git_repo=https://github.com/tyayers/apigee-aigateway-lab&cloudshell_git_branch=main&cloudshell_workspace=.&cloudshell_tutorial=TUTORIAL.md)

Or direct [markdown](https://github.com/tyayers/ai-gateway-labs/blob/main/TUTORIAL.md) link.
