#!/bin/bash

# Navega para o diretório raiz do projeto
cd "$(dirname "$0")/.."

echo -e "\033[33mConfigurando Namespace...\033[0m"
kubectl apply -f k8s/local/namespace.yaml

echo "Aplicando ConfigMaps e Secrets..."
kubectl apply -f k8s/local/configmap.yaml
kubectl apply -f k8s/local/secrets.yaml
kubectl apply -f k8s/local/postgres-init.yaml

echo "Iniciando Infraestrutura Base..."
kubectl apply -f k8s/local/infra/postgres.yaml
kubectl apply -f k8s/local/infra/redis.yaml
kubectl apply -f k8s/local/infra/rabbitmq.yaml
kubectl apply -f k8s/local/infra/minio.yaml
kubectl apply -f k8s/local/infra/prometheus.yaml
kubectl apply -f k8s/local/infra/grafana.yaml
kubectl apply -f k8s/local/infra/sonarqube.yaml
kubectl apply -f k8s/local/infra/metabase.yaml

echo "Aguardando inicialização da infraestrutura (RabbitMQ, Postgres, etc)..."
kubectl rollout status deployment/rabbitmq -n g57 --timeout=120s
kubectl rollout status deployment/postgres -n g57 --timeout=120s
kubectl rollout status deployment/redis -n g57 --timeout=120s
kubectl rollout status deployment/minio -n g57 --timeout=120s

echo "Iniciando Microsserviços..."
kubectl apply -f k8s/local/services/auth-service.yaml
kubectl apply -f k8s/local/services/video-service.yaml
kubectl apply -f k8s/local/services/processing-service.yaml
kubectl apply -f k8s/local/services/status-service.yaml
kubectl apply -f k8s/local/services/notification-service.yaml
kubectl apply -f k8s/local/services/api-gateway.yaml
kubectl apply -f k8s/local/services/frontend.yaml

read -p "Deseja buildar as imagens localmente? (S/N) " response
if [[ "$response" =~ ^[Ss]$ ]]; then
    echo "Buildando Auth Service..."
    minikube image build -t g57-auth-service:latest ./services/auth-service
    
    echo "Buildando Video Service..."
    minikube image build -t g57-video-service:latest ./services/video-service
    
    echo "Buildando Processing Service..."
    minikube image build -t g57-processing-service:latest ./services/processing-service
    
    echo "Buildando Status Service..."
    minikube image build -t g57-status-service:latest ./services/status-service
    
    echo "Buildando Notification Service..."
    minikube image build -t g57-notification-service:latest ./services/notification-service
    
    echo "Buildando API Gateway..."
    minikube image build -t g57-api-gateway:latest ./services/api-gateway
    
    echo "Buildando Frontend..."
    docker build -t g57-frontend:latest ./frontend
    minikube image load g57-frontend:latest
    
    kubectl rollout restart deployment -n g57
fi

echo -e "\033[32mDeploy concluído! Verifique os pods com: kubectl get pods -n g57\033[0m"

echo ""
echo -e "\033[36mConfigurando Port-Forwarding...\033[0m"
echo "Iniciando port-forward do API Gateway (8080) em background..."
kubectl port-forward service/api-gateway 8080:8080 -n g57 &

echo ""
echo -e "\033[36mPara acessar o Frontend, execute: minikube service frontend -n g57\033[0m"
