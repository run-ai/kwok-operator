#! /bin/bash
if [ -z "$1" ]; then
  echo "There is an option to install metrics-server."
  echo "run 'bash install_kwok.sh true' to install metrics-server. by default, it is false."
  INSTALL_METRICS_SERVER=false
else
  INSTALL_METRICS_SERVER=$1
fi


KWOK_REPO=kubernetes-sigs/kwok
KWOK_LATEST_RELEASE=$(curl "https://api.github.com/repos/${KWOK_REPO}/releases/latest" | jq -r '.tag_name')

echo "Apply Kwok version ${KWOK_LATEST_RELEASE}..."
kubectl apply -f "https://github.com/${KWOK_REPO}/releases/download/${KWOK_LATEST_RELEASE}/kwok.yaml"
echo "check Kwok status..."
kubectl wait --for=condition=Ready -n kube-system pod -l app=kwok-controller --timeout=300s
if [ $? -ne 0 ]; then
  echo "Kwok is not ready, please check the logs."
  exit 1
fi
echo "Kwok is ready!"
echo "install CRDs for Kwok..."
kubectl apply -f "https://github.com/${KWOK_REPO}/releases/download/${KWOK_LATEST_RELEASE}/stage-fast.yaml"

if [ "$INSTALL_METRICS_SERVER" = true ]; then
  echo "install metrics-server..."
  kubectl apply -f "https://github.com/${KWOK_REPO}/releases/download/${KWOK_LATEST_RELEASE}/metrics-usage.yaml"
fi

echo "Kwok is installed successfully! - Happy Kwoking!"