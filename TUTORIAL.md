# Apigee AI Gateway Lab
---
This tutorial helps you to provision an Apigee AI Gateway into your Google Cloud project, and create some AI proxies to add governance to tools like Gemini CLI and Claude Code.

Let's get started!

---

## Prerequisites

The only prerequisite for this lab is that you have a Google Cloud project to use.

These [Google Cloud roles](https://docs.cloud.google.com/iam/docs/roles-permissions) are needed by your user for the deployments:

* Apigee Organization Admin (roles/apigee.admin)
* API Hub Admin(roles/apihub.admin)
* Service Usage Admin (roles/serviceusage.serviceUsageAdmin)
* Service Account Admin (roles/iam.serviceAccountAdmin)
* Compute Admin (roles/compute.admin)
* Compute Network Admin (roles/compute.networkAdmin)
* Cloud KMS Admin (roles/cloudkms.admin)
* Agent Platform Admin (roles/ml.admin)

---

## Setup

You will need to set these environment variables to run this lab.

| Variable | Description |
| -------- | ----------- |
| GOOGLE_CLOUD_PROJECT | Your Google Cloud project id |
| GOOGLE_CLOUD_LOCATION | Your Google Cloud region for the Apigee region |
| APIGEE_TYPE | The type of Apigee deployment (either EVALUATION (valid for 60 days), PAYG (consumption pricing, path to production), or SUBSCRIPTION (fixed pricing)) |

Additionally these optional variables can be set if you want to use an existing VPC and subnet, or use a DRZ data residency location.

| Optional Variable | Description |
| -------- | ----------- |
| APIGEE_VPC_NAME | The name of your existing VPC to use for Apigee |
| APIGEE_SUBNET_NAME | The name of your existing VPC subnet to use for Apigee |
| APIGEE_DRZ_LOCATION | The optional DRZ data residency location for Apigee data (US, EU or IN) |

To set these, you can easily copy the `env.sh` file to a local `.env` file.

```sh
cp env.sh .env
```

Click  <walkthrough-editor-open-file filePath=".env">here</walkthrough-editor-open-file> to open the `.env` file in the editor.

Set your values, save the file, and then run the `source .env`.

```sh
source .env
```

### Install tooling
This lab uses two open source CLIs to automate Apigee, [apigeecli](https://github.com/apigee/apigeecli) and [aft](https://github.com/apigee/apigee-templater).

If not already installed, install them into your shell.

```sh
curl -L https://raw.githubusercontent.com/apigee/apigeecli/main/downloadLatest.sh | sh -

npm i apigee-templater -g
```

---

## Provision Apigee

You can provision your Apigee instance in any of the ways documented [here](https://docs.cloud.google.com/apigee/docs/api-platform/get-started/provisioning-options).

If you already have Apigee provisioned, then you can skip this step.

For a simple automated deployment, we can run the sample Terraform deployment in this lab, which also creates a load balancer and test certificate to access the instance.

Take a look at the <walkthrough-editor-open-file filePath="tf/apigee/main.tf">main.tf</walkthrough-editor-open-file> file to see the resources created.

To provision the lab sample Apigee instance, run these commands now.

### Provision resources

The included Terraform template in this lab can easily provision all resources in an empty Google Cloud project with reasonable defaults (default network, Apigee evaluation, Model Garden, Load Balancer & Certificates, etc..).

If you want to adjust the defaults, you can add these variables to the `apply` command below: ` --var "drz_location=$APIGEE_DRZ_LOCATION" --var "apigee_type=$APIGEE_TYPE" --var "network=$APIGEE_VPC_NAME" --var "subnet=$APIGEE_SUBNET_NAME"`.

```sh
cd tf/apigee
terraform init
terraform apply -var "project_id=$GOOGLE_CLOUD_PROJECT" -var "region=$GOOGLE_CLOUD_LOCATION" --var "apigee_type=$APIGEE_TYPE"
cd ../..
```

Provisioning takes around 20-30 minutes for Apigee, Model Garden, Load Balancer, Network Services, etc... to be deployed.

---

## Initialize environment

After provisioning is finished, let's initialize some Apigee environment variables and create the analytics data collectors for AI proxies.

Take a look at the <walkthrough-editor-open-file filePath="script_initialize.sh">script_initialize.sh</walkthrough-editor-open-file> file to see the commands that are run.

```sh
source script_initialize.sh
```

---

## Test Gemini API

Now let's test if the Gemini API on [Gemini Enterprise Agent Platfrom](https://docs.cloud.google.com/gemini-enterprise-agent-platform) is working.

```sh
curl -i -X POST "https://aiplatform.googleapis.com/v1/projects/$GOOGLE_CLOUD_PROJECT/locations/global/publishers/google/models/gemini-flash-latest:generateContent" \
-H "Authorization: Bearer $(gcloud auth application-default print-access-token)" \
-H "Content-Type: application/json" \
-d '{"contents": [{"role": "USER", "parts": [{"text": "why is the sky blue?"}]}]}'
```

You should get a response with an answer candidate with some text about 'Rayleigh scattering'. 

---

## Deploy Gemini Proxy

Create a simple **AI-Gemini** proxy using the `aft` command with a base path of **/gemini** and deploying it to a proxy called **AI-Gemini** in your Apigee environment.

```sh
aft -b /gemini -u https://aiplatform.googleapis.com -o $GOOGLE_CLOUD_PROJECT:AI-Gemini:$APIGEE_ENVIRONMENT
```

Open the proxy in the [Google Cloud Console](https://console.cloud.google.com/apigee/proxies/AI-Gemini/overview), and wait until the deployment is complete (you should see a gree ✅ next to the deployment).

After the deployment is complete, click on the **Debug** tab in the proxy screen, and start a debug session.

Let's now call the proxy URL with our same prompt, but this time see the request processing through our proxy in Apigee. Notice the **$APIGEE_HOST** parameter in the URL, which points the request to our Apigee endpoint.

```sh
curl -i -X POST "https://$APIGEE_HOST/gemini/v1/projects/$GOOGLE_CLOUD_PROJECT/locations/global/publishers/google/models/gemini-flash-latest:generateContent" \
-H "Authorization: Bearer $(gcloud auth application-default print-access-token)" \
-H "Content-Type: application/json" \
-d '{"contents": [{"role": "USER", "parts": [{"text": "why is the sky blue?"}]}]}'
```

You should get a similar response again about 'Rayleigh scattering'. 

Go back to the Debug panel, and see the processing steps, timings and variables that were done between the request and response.

---

## Add Model Authorization, Governance & Analytics Features

Now we will update the proxy for **Gemini**, and also add a new one for **Claude**, with a template that applies model authorization & governance.

Open the template file <walkthrough-editor-open-file filePath="AI-Proxy-Gemini.yaml">AI-Proxy-Gemini.yaml</walkthrough-editor-open-file> to see how a proxy template is structured, including the parameters (where we are setting a base path of **/gemini** and a model filter to only allow **gemini** models).

```sh
aft AI-Proxy-Gemini.yaml -o $GOOGLE_CLOUD_PROJECT:AI-Gemini:$APIGEE_ENVIRONMENT:ai-service@$GOOGLE_CLOUD_PROJECT.iam.gserviceaccount.com
aft AI-Proxy-Claude.yaml -o $GOOGLE_CLOUD_PROJECT:AI-Claude:$APIGEE_ENVIRONMENT:ai-service@$GOOGLE_CLOUD_PROJECT.iam.gserviceaccount.com

aft -i AI-Analytics.yaml -o $GOOGLE_CLOUD_PROJECT:AI-Analytics:$APIGEE_ENVIRONMENT
```

Create a product & subscription to the **AI-Gemini** proxy.

Take a look at the <walkthrough-editor-open-file filePath="script_register_key.sh">script_register_key.sh</walkthrough-editor-open-file> file to see the commands that are run.

```sh
source script_register_key.sh
```

---

## Test Model Proxy Authorization & Failover

Now let's call our model proxy with an API key as credential, that has subscribed to the **AI-Gemini** product with certain LLM token quotas.

```sh
curl -i -X POST "https://$APIGEE_HOST/gemini/v1/projects/$GOOGLE_CLOUD_PROJECT/locations/global/publishers/google/models/gemini-flash-latest:generateContent" \
-H "x-api-key: $API_KEY" \
-H "Content-Type: application/json" \
-d '{"contents": [{"role": "USER", "parts": [{"text": "why is the sky blue?"}]}]}'
```

## View Analytics Data

TODO

---

## Conclusion
<walkthrough-conclusion-trophy></walkthrough-conclusion-trophy>

Congratulations! You've successfully deployed Apigee AI proxies into your Google Cloud project, as a next step try expanding the proxies to more models, product definitions, and tools such as API & MCP servers.
<walkthrough-inline-feedback></walkthrough-inline-feedback>
