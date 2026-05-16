# import products
apigeecli products import -f AI-Gemini-Product.json -o $GOOGLE_CLOUD_PROJECT --default-token

# create test developer
apigeecli developers create -n "test@example.com" \
  -f "Test" -s "Developer" \
  -u "test@example.com" -o $GOOGLE_CLOUD_PROJECT --default-token

# create app and get key
export API_KEY=$(apigeecli apps create --name "Gemini App" \
  --email "test@example.com" \
  --prods "Gemini Product" \
  --org $GOOGLE_CLOUD_PROJECT --default-token | jq --raw-output '.credentials[0].consumerKey')

echo "Your API key to access the Gemini Product is: $API_KEY"
