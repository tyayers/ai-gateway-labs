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

To set these, you can easily copy the `env.sh` file to a local `.env` file.

```sh
cp env.sh .env
```

Click  <walkthrough-editor-open-file filePath=".env">here</walkthrough-editor-open-file> to open the `.env` file in the editor.

Set your values, save the file, and then run the `source .env`.

```sh
source .env
```

---

## Provision Apigee

You can provision your Apigee instance in any of the ways documented [here](https://docs.cloud.google.com/apigee/docs/api-platform/get-started/provisioning-options).

For a simple **TRIAL** deployment, we can run the sample Terraform deployment in this lab, which also creates a load balancer and test certificate to access the instance.

Take a look at the <walkthrough-editor-open-file filePath="tf/provision/main.tf">main.tf</walkthrough-editor-open-file> file to see the resources created.

To provision the lab sample Apigee instance, run these commands now.

### Provision global control plane with default VPC

```sh
cd tf/provision
terraform init
terraform apply -var "project_id=$GOOGLE_CLOUD_PROJECT" -var "region=$GOOGLE_CLOUD_LOCATION" --var "apigee_type=$APIGEE_TYPE"
cd ../..
```

### Provision regional EU control plane with custom VPC / subnet

```sh
cd tf/provision-drz
terraform init
terraform apply -var "project_id=$GOOGLE_CLOUD_PROJECT" -var "region=$GOOGLE_CLOUD_LOCATION" --var "apigee_type=$APIGEE_TYPE" --var "network=YOUR_NETWORK" --var "subnet=YOUR_SUBNET"
cd ../..
```

Provisioning takes between 30-60 minutes for Apigee, API Hub, a Global Load Balancer, Certificates, etc.. to be completed.

### Save Apigee environment information

After it's finished, or if you have your own Apigee instance, let's install [apigeecli](https://github.com/apigee/apigeecli) and save some basic information into environment variables.

```sh
curl -L https://raw.githubusercontent.com/apigee/apigeecli/main/downloadLatest.sh | sh -

APIGEE_ENVIRONMENT=$(apigeecli environments list -o $GOOGLE_CLOUD_PROJECT --default-token | jq --raw-output '.[0]')
echo "Your Apigee environment is: $APIGEE_ENVIRONMENT"
APIGEE_HOST=$(apigeecli envgroups list -o $GOOGLE_CLOUD_PROJECT --default-token | jq --raw-output '.environmentGroups[0].hostnames[-1]')
echo "Your Apigee host is: $APIGEE_HOST"
```

### Create analytics data collectors

In Apigee we can flexibly define which analytics data properties we want to collect from the runtime traffic, so lets create several for our AI proxies.

```sh
apigeecli datacollectors create -d "Model name" -n dc_ai_model -p STRING --org "$GOOGLE_CLOUD_PROJECT" --default-token
apigeecli datacollectors create -d "Model cost center" -n dc_ai_cost_center -p STRING --org "$GOOGLE_CLOUD_PROJECT" --default-token
apigeecli datacollectors create -d "Total token count" -n dc_ai_total_token_count -p INTEGER --org "$GOOGLE_CLOUD_PROJECT" --default-token
apigeecli datacollectors create -d "Prompt token count" -n dc_ai_prompt_token_count -p INTEGER --org "$GOOGLE_CLOUD_PROJECT" --default-token
apigeecli datacollectors create -d "Total token count" -n dc_ai_response_token_count -p INTEGER --org "$GOOGLE_CLOUD_PROJECT" --default-token
apigeecli datacollectors create -d "Model response type" -n dc_ai_response_type -p STRING --org "$GOOGLE_CLOUD_PROJECT" --default-token
apigeecli datacollectors create -d "Time to first token" -n dc_ai_time_first_token -p INTEGER --org "$GOOGLE_CLOUD_PROJECT" --default-token
```

### Create AI service account

Let's also enable the Agent Platform in Google Cloud, and create a service account user, as well as assign our user as token creator.

```sh
gcloud services enable aiplatform.googleapis.com --project $GOOGLE_CLOUD_PROJECT

gcloud iam service-accounts create "ai-service" --project="$GOOGLE_CLOUD_PROJECT" \
    --description="AI service account" \
    --display-name="API Service Account"
gcloud projects add-iam-policy-binding $GOOGLE_CLOUD_PROJECT \
    --member="serviceAccount:ai-service@$GOOGLE_CLOUD_PROJECT.iam.gserviceaccount.com" \
    --role="roles/aiplatform.user"

gcloud iam service-accounts add-iam-policy-binding \
  ai-service@$GOOGLE_CLOUD_PROJECT.iam.gserviceaccount.com \
  --member="user:$(gcloud config get-value account 2>/dev/null)" \
  --role="roles/iam.serviceAccountTokenCreator"

PROJECT_NUMBER=$(gcloud projects describe $GOOGLE_CLOUD_PROJECT --format="value(projectNumber)")
gcloud iam service-accounts add-iam-policy-binding \
  ai-service@$GOOGLE_CLOUD_PROJECT.iam.gserviceaccount.com \
  --member="serviceAccount:service-$PROJECT_NUMBER@gcp-sa-apigee.iam.gserviceaccount.com" \
  --role="roles/iam.serviceAccountTokenCreator" --project $GOOGLE_CLOUD_PROJECT
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

## Deploy Gemini & Analytics Proxy

Now let's install the tool [Apigee Feature Templater](https://github.com/apigee/apigee-templater), which makes it simple to create and deploy templated proxies.

```sh
npm i apigee-templater -g
```

Open the Apigee proxy template file <walkthrough-editor-open-file filePath="AI-Gemini.yaml">AI-Gemini.yaml</walkthrough-editor-open-file> to see how the template is structured.

It references a feature called <walkthrough-editor-open-file filePath="AI-Proxy-AgentPlatform.yaml">AI-Proxy-AgentPlatform.yaml</walkthrough-editor-open-file>, which has the proxy configuration to Agent Platform, but only allowing Gemini models through the **AllowedModelPatterns** property.

### Deploy Model Proxy Template

```sh
aft -i AI-Gemini.yaml -o $GOOGLE_CLOUD_PROJECT:AI-Gemini:$APIGEE_ENVIRONMENT:ai-service@$GOOGLE_CLOUD_PROJECT.iam.gserviceaccount.com
```

### Deploy Analytics Proxy Template

Let's deploy an extra analytics proxy that will be used to collect and save all token analytics data, from any model and proxy.

```sh
aft -i AI-Analytics.yaml -o $GOOGLE_CLOUD_PROJECT:AI-Analytics:$APIGEE_ENVIRONMENT
```

Now open both deployed proxies in the [Apigee console](https://console.cloud.google.com/apigee/proxies) and take a look at the configuration there.

---
## Configure a Product to access Gemini

Import a [product definition](https://docs.cloud.google.com/apigee/docs/api-platform/publish/what-api-product) to manage authorization & access to models and token quotas.

```sh
apigeecli products import -o $GOOGLE_CLOUD_PROJECT -f AI-Gemini-Product.json --default-token
```

Open the product definition in the [Apigee console](https://console.cloud.google.com/apigee/apiproducts) and take a look at the configuration. Notice how there is a token quota on **gemini-flash-latest**.

Go to the **Developers** and **Apps** pages. Register a developer and an app, and get a **Credential Key** that can be used to access the proxy endpoint.

Save the key in an environment variable.

```sh
API_KEY=YOUR_KEY
```

---

## Call the Gemini proxy

Call the Gemini proxy endpoint with the same prompt as before. Notice the **/gemini** in the path below, which routes the traffic through our proxy.

```sh
curl -i -X POST "https://$APIGEE_HOST/gemini/v1/projects/$GOOGLE_CLOUD_PROJECT/locations/global/publishers/google/models/gemini-flash-latest:generateContent" \
-H "Authorization: Bearer $(gcloud auth application-default print-access-token)" \
-H "Content-Type: application/json" \
-d '{"contents": [{"role": "USER", "parts": [{"text": "why is the sky blue?"}]}]}'
```

You should get a **401** unauthorized response. Now let's add our API key, and remove the Google Cloud credentials.

Run the same call with our API key:

```sh
curl -i -X POST "https://$APIGEE_HOST/gemini/v1/projects/$GOOGLE_CLOUD_PROJECT/locations/global/publishers/google/models/gemini-flash-latest:generateContent" \
-H "x-api-key: $API_KEY" \
-H "Content-Type: application/json" \
-d '{"contents": [{"role": "USER", "parts": [{"text": "why is the sky blue?"}]}]}'
```

You should get again the response with **bold**

---

## Conclusion
<walkthrough-conclusion-trophy></walkthrough-conclusion-trophy>

Congratulations! You've successfully deployed Apigee AI proxies into your Google Cloud project, as a next step try expanding the proxies to more models, product definitions, and tools such as API & MCP servers.
<walkthrough-inline-feedback></walkthrough-inline-feedback>
