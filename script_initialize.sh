# get environment variables
export APIGEE_ENVIRONMENT=$(apigeecli environments list -o $GOOGLE_CLOUD_PROJECT --default-token | jq --raw-output '.[0]')
echo "Your Apigee environment is: $APIGEE_ENVIRONMENT"
export APIGEE_HOST=$(apigeecli envgroups list -o $GOOGLE_CLOUD_PROJECT --default-token | jq --raw-output '.environmentGroups[0].hostnames[-1]')
echo "Your Apigee host is: $APIGEE_HOST"

# create data collectors
apigeecli datacollectors create -d "Model name" -n dc_ai_model -p STRING --org "$GOOGLE_CLOUD_PROJECT" --default-token
apigeecli datacollectors create -d "Model cost center" -n dc_ai_cost_center -p STRING --org "$GOOGLE_CLOUD_PROJECT" --default-token
apigeecli datacollectors create -d "Total token count" -n dc_ai_total_token_count -p INTEGER --org "$GOOGLE_CLOUD_PROJECT" --default-token
apigeecli datacollectors create -d "Prompt token count" -n dc_ai_prompt_token_count -p INTEGER --org "$GOOGLE_CLOUD_PROJECT" --default-token
apigeecli datacollectors create -d "Total token count" -n dc_ai_response_token_count -p INTEGER --org "$GOOGLE_CLOUD_PROJECT" --default-token
apigeecli datacollectors create -d "Model response type" -n dc_ai_response_type -p STRING --org "$GOOGLE_CLOUD_PROJECT" --default-token
apigeecli datacollectors create -d "Time to first token" -n dc_ai_time_first_token -p INTEGER --org "$GOOGLE_CLOUD_PROJECT" --default-token

# create AI service account
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
  --role="roles/iam.serviceAccountTokenCreator" --project $GOOGLE_CLOUD_PROJECT

PROJECT_NUMBER=$(gcloud projects describe $GOOGLE_CLOUD_PROJECT --format="value(projectNumber)")
gcloud iam service-accounts add-iam-policy-binding \
  ai-service@$GOOGLE_CLOUD_PROJECT.iam.gserviceaccount.com \
  --member="serviceAccount:service-$PROJECT_NUMBER@gcp-sa-apigee.iam.gserviceaccount.com" \
  --role="roles/iam.serviceAccountTokenCreator" --project $GOOGLE_CLOUD_PROJECT
