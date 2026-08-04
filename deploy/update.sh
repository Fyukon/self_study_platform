#!/bin/sh
# Deploy the Learning Roadmap app on the GerPRO server.
#
# Server layout:
#   /opt/learning-roadmap/repo                  git clone of this repository
#   /opt/learning-roadmap/learning-roadmap      installed production binary
#   /var/lib/learning-roadmap                   runtime data (XDG_DATA_HOME=/var/lib)
#   /root/.ssh/learning-roadmap_deploy_ed25519  read-only GitHub deploy key
#
# Usage: ./deploy/update.sh [VERSION]

set -eu

APP=/opt/learning-roadmap/learning-roadmap
REPO=/opt/learning-roadmap/repo
KEY=/root/.ssh/learning-roadmap_deploy_ed25519
REMOTE=git@github.com:Fyukon/self_study_platform.git
VERSION="${1:-}"

if [ ! -d "$REPO/.git" ]; then
  mkdir -p "$(dirname "$REPO")"
  GIT_SSH_COMMAND="ssh -i $KEY -o IdentitiesOnly=yes" git clone "$REMOTE" "$REPO"
fi

cd "$REPO"
GIT_SSH_COMMAND="ssh -i $KEY -o IdentitiesOnly=yes" git fetch --tags origin
GIT_SSH_COMMAND="ssh -i $KEY -o IdentitiesOnly=yes" git reset --hard "@{upstream}"

if [ -z "$VERSION" ]; then
  VERSION=$(git describe --tags --always 2>/dev/null || echo dev)
fi

make build VERSION="$VERSION"
install -o root -g root -m 755 dist/learning-roadmap "$APP"
systemctl restart learning-roadmap
