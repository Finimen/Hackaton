# Build agent
docker build -t netscan-agent -f deployments/agent/Dockerfile .

# Run into the remote server
export AGENT_NAME="my-agent"
export AGENT_LOCATION="datacenter-1" 
export BACKEND_URL="https://api.netscan.com"
docker run -d --name netscan-agent \
  -e AGENT_NAME=$AGENT_NAME \
  -e AGENT_LOCATION=$AGENT_LOCATION \
  -e BACKEND_URL=$BACKEND_URL \
  --network host \
  netscan-agent