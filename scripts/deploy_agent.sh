# Example 1
# ./deploy-agent.sh "digitalocean-nyc" "usa-nyc" "https://backend.netscan.com"
# Example 2
# ./deploy-agent.sh "aws-london" "uk-london" "https://backend.netscan.com" "existing-token-here"

AGENT_NAME=$1
LOCATION=$2
BACKEND_URL=$3
AGENT_TOKEN=$4

if [ -z "$AGENT_NAME" ] || [ -z "$LOCATION" ] || [ -z "$BACKEND_URL" ]; then
    echo "Usage: ./deploy-agent.sh <agent-name> <location> <backend-url> [agent-token]"
    echo "Example: ./deploy-agent.sh aws-us-east usa-east-1 https://api.netscan.com"
    exit 1
fi

# Собираем и запускаем агента
cd deployments/agent

AGENT_NAME=$AGENT_NAME \
AGENT_LOCATION=$LOCATION \
BACKEND_URL=$BACKEND_URL \
AGENT_TOKEN=$AGENT_TOKEN \
docker-compose -f docker-compose.agent.yml up -d

echo "✅ Agent $AGENT_NAME deployed successfully!"
echo "📊 Backend: $BACKEND_URL"
echo "📍 Location: $LOCATION"