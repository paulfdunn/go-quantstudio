#!/bin/zsh
colima start

# PRIOR TO EXECUTING THIS SCRIPT
# You have to delete all prior containers/images, otherwise the build fails with:
# internal/syscall/execenv: /usr/local/go/pkg/tool/linux_amd64/compile: signal: segmentation fault (core dumped)
docker container prune
docker image prune -a

# Build and push container to Artifact Registry
export GCLOUD_PROJECT="go-quantstudio-new-430921"
export REPO="go-quantstudio-new-repo"
export REGION="us-west3"
export IMAGE="go-quantstudio"
export IMAGE_TAG=${REGION}-docker.pkg.dev/$GCLOUD_PROJECT/$REPO/$IMAGE

docker build -t $IMAGE_TAG --platform linux/amd64 . && docker push $IMAGE_TAG