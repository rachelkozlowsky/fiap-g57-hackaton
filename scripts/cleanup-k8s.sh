#!/bin/bash

echo -e "\033[33mLimpando ambiente Kubernetes...\033[0m"

echo "Deletando PVCs (volumes persistentes)..."
kubectl delete pvc --all -n g57 --ignore-not-found=true

echo "Deletando namespace g57..."
kubectl delete namespace g57 --ignore-not-found=true

echo "Aguardando exclusão completa do namespace..."
sleep 10

read -p "Deseja limpar as imagens Docker do Minikube? (S/N) " response
if [[ "$response" =~ ^[Ss]$ ]]; then
    echo "Limpando imagens antigas..."
    minikube image rm g57-auth-service:latest
    minikube image rm g57-video-service:latest
    minikube image rm g57-processing-service:latest
    minikube image rm g57-notification-service:latest
    minikube image rm g57-status-service:latest
    minikube image rm g57-api-gateway:latest
    minikube image rm g57-frontend:latest
fi

echo -e "\033[32mLimpeza concluída!\033[0m"
echo ""
echo -e "\033[36mPara reiniciar do zero, execute:\033[0m"
echo -e "  ./scripts/deploy-k8s.sh"
