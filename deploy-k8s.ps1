
# Script de deploy completo para Windows (PowerShell)

# 1. Configurar Namespace
Write-Host "Configurando Namespace..."
kubectl apply -f k8s/namespace.yaml

# 2. Configurar ConfigMaps e Secrets
Write-Host "Aplicando ConfigMaps e Secrets..."
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/secrets.yaml
kubectl apply -f k8s/postgres-init.yaml

# 3. Deploy da Infraestrutura Base (Banco de Dados, Filas, Storage)
Write-Host "Iniciando Infraestrutura Base..."
kubectl apply -f k8s/infra/postgres.yaml
kubectl apply -f k8s/infra/redis.yaml
kubectl apply -f k8s/infra/rabbitmq.yaml
kubectl apply -f k8s/infra/minio.yaml

# Aguardar RabbitMQ e outras infraestruturas estarem prontas
Write-Host "Aguardando inicialização da infraestrutura (RabbitMQ, Postgres, etc)..."
kubectl rollout status deployment/rabbitmq -n g57 --timeout=120s
kubectl rollout status deployment/postgres -n g57 --timeout=120s
kubectl rollout status deployment/redis -n g57 --timeout=120s
kubectl rollout status deployment/minio -n g57 --timeout=120s

# 4. Deploy dos Microsserviços
Write-Host "Iniciando Microsserviços..."
kubectl apply -f k8s/services/auth-service.yaml
kubectl apply -f k8s/services/video-service.yaml
kubectl apply -f k8s/services/processing-service.yaml
kubectl apply -f k8s/services/status-service.yaml
kubectl apply -f k8s/services/notification-service.yaml
kubectl apply -f k8s/services/api-gateway.yaml
kubectl apply -f k8s/services/frontend.yaml

# 5. Build e Load das imagens (caso não estejam no registry)
# Nota: Isso assume que você quer buildar localmente. Se já tiver as imagens, pode pular.
Write-Host "Deseja buildar as imagens localmente? (S/N)"
$response = Read-Host
if ($response -eq 'S' -or $response -eq 's') {
    Write-Host "Buildando Auth Service..."
    minikube image build -t g57-auth-service:latest ./services/auth-service
    
    Write-Host "Buildando Video Service..."
    minikube image build -t g57-video-service:latest ./services/video-service
    
    Write-Host "Buildando Processing Service..."
    minikube image build -t g57-processing-service:latest ./services/processing-service
    
    Write-Host "Buildando Status Service..."
    minikube image build -t g57-status-service:latest ./services/status-service
    
    Write-Host "Buildando Notification Service..."
    minikube image build -t g57-notification-service:latest ./services/notification-service
    
    Write-Host "Buildando API Gateway..."
    minikube image build -t g57-api-gateway:latest ./services/api-gateway
    
    Write-Host "Buildando Frontend..."
    # Workaround para evitar erros de contexto no minikube image build
    docker build -t g57-frontend:latest ./frontend
    minikube image load g57-frontend:latest
    
    # Reiniciar deployments para pegar as novas imagens
    kubectl rollout restart deployment -n g57
}

Write-Host "Deploy concluído! Verifique os pods com: kubectl get pods -n g57"

Write-Host "Configurando Port-Forwarding..."
Write-Host "Abrindo novo terminal para API Gateway (8080)..."
Start-Process powershell -ArgumentList "-NoExit", "-Command", "kubectl port-forward service/api-gateway 8080:8080 -n g57"

Write-Host "Para acessar o Frontend, execute em outro terminal: minikube service frontend -n g57"

