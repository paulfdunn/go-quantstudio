#!/bin/zsh
# Build and push container to Artifact Registry
colima start
export GCLOUD_PROJECT="go-quantstudio-new-430921"
export REPO="go-quantstudio-new-repo"
export REGION="us-west3"
export IMAGE="go-quantstudio"
export IMAGE_TAG=${REGION}-docker.pkg.dev/$GCLOUD_PROJECT/$REPO/$IMAGE

docker build -t $IMAGE_TAG --platform linux/x86_64 .
docker push $IMAGE_TAG