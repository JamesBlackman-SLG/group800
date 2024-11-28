docker-compose build
docker-compose push
# Use the environment variables directly
APP_ID="$DIGITAL_OCEAN_800_GROUP_APP_ID"
API_TOKEN="$DIGITAL_OCEAN_API_TOKEN"

# Trigger the redeployment
curl -X POST -d '{}' "https://api.digitalocean.com/v2/apps/$APP_ID/deployments" \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json"
