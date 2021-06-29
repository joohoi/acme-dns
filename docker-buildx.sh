# Check buildx:
#   docker buildx ls
#
# Should have output like: 
#    default  default running linux/amd64, linux/386, linux/arm64, linux/riscv64, linux/ppc64le, linux/s390x, linux/arm/v7, linux/arm/v6             
#     
# If not, to setup buildx (if you are using docker desktop) and buildx command is there:
#    make sure in:                 $USER/.docker/config.json
#    (may not exist), you have:    {"experimental": "enabled"}
# If you see "buildx command not recognized":
#   export DOCKER_BUILDKIT=1 && docker build --platform=local -o . git://github.com/docker/buildx && mkdir -p ~/.docker/cli-plugins && mv buildx ~/.docker/cli-plugins/docker-buildx
#
# But normally let's say you start a brand new digitalocean droplet for builds, and you just do apt install -y docker.io or use the prebuilt docker, I've had to do the following:
#   docker run --rm --privileged multiarch/qemu-user-static --reset -p yes && docker buildx rm builder && docker buildx create --name builder --driver docker-container --use && docker buildx inspect --bootstrap
# 
#  
# Usage (assuming you have buildx, docker 18+):
#   ./docker-buildx vX.X.X

version=$1
platforms="linux/arm64,linux/amd64"

echo "Building $version of acme-dns for platfroms: $platforms!"

docker buildx build \
    --platform $platforms \
    --rm \
    --push \
    --compress \
    -t joohoi/acme-dns:$version \
    -f ./Dockerfile.buildx .