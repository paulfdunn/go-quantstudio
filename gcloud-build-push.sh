#!/bin/zsh
colima stop
colima start

# You have to delete all prior containers/images, otherwise the build fails with:
# internal/syscall/execenv: /usr/local/go/pkg/tool/linux_amd64/compile: signal: segmentation fault (core dumped)
# Actually the build just intermittently fails - retry if it fails for the above.
docker container prune -f
docker image prune -a -f

# Build and push container to Artifact Registry
export GCLOUD_PROJECT="go-quantstudio-new-430921"
export REPO="go-quantstudio-new-repo"
export REGION="us-west3"
export IMAGE="go-quantstudio"
export IMAGE_TAG=${REGION}-docker.pkg.dev/$GCLOUD_PROJECT/$REPO/$IMAGE

docker build -t $IMAGE_TAG --platform linux/amd64 . && docker push $IMAGE_TAG

# Go to the artifact registry: https://console.cloud.google.com/artifacts?hl=en&inv=1&invt=AbtGMw&project=go-quantstudio-new-430921
# Select:  go-quantstudio-new-repo
# Select:  go-quantstudio
# Delete the old version
# Go to Cloud Run Services: https://console.cloud.google.com/run?hl=en&invt=AbtGNQ&project=go-quantstudio-new-430921
# Select: go-quantstudio
# Select: Edit and deploy new revision
# Select: Container image URL and select the 'latest' revision.
# Select: deploy