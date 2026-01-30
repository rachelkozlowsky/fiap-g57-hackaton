# Guia de Implantação Kubernetes (Minikube) - Hackathon G57

Este guia descreve os passos para subir toda a infraestrutura e microsserviços do projeto no Kubernetes local utilizando Minikube.

## 📋 Pré-requisitos

*   **Docker** instalado e rodando.
*   **Minikube** instalado.
*   **Kubectl** instalado.
*   **PowerShell** (para executar o script de automação no Windows).

## 🚀 1. Iniciar o Minikube

Antes de qualquer comando, certifique-se de que o cluster Minikube está ativo. Se você recebeu erros de conexão (`connectex: No connection could be made`), provavelmente o Minikube está parado.

```powershell
minikube start --driver=docker
```

Habilite os addons necessários (caso ainda não tenha feito):
```powershell
minikube addons enable ingress
minikube addons enable metrics-server
minikube addons enable dashboard
```

## 🛠️ 2. Instalação Automática (Recomendado)

Foi criado um script PowerShell que automatiza todo o processo de deploy, incluindo a criação de namespaces, secrets, infraestrutura e serviços.

Execute no terminal:

```powershell
.\deploy-k8s.ps1
```

O script irá:
1.  Criar o namespace `g57`.
2.  Aplicar ConfigMaps e Secrets.
3.  Subir Postgres, Redis, RabbitMQ e MinIO.
4.  Subir todos os microsserviços.
5.  Perguntar se deseja rebuildar as imagens Docker localmente.
6.  Ao final, abrir um túnel para o API Gateway automaticamente.

## 📦 3. Instalação Manual (Passo a Passo)

Caso prefira executar comando por comando:

### 3.1 Namespace e Configurações
```powershell
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/secrets.yaml
kubectl apply -f k8s/postgres-init.yaml
```

### 3.2 Infraestrutura Base
```powershell
kubectl apply -f k8s/infra/postgres.yaml
kubectl apply -f k8s/infra/redis.yaml
kubectl apply -f k8s/infra/rabbitmq.yaml
kubectl apply -f k8s/infra/minio.yaml
```

### 3.3 Microsserviços
```powershell
kubectl apply -f k8s/services/auth-service.yaml
kubectl apply -f k8s/services/video-service.yaml
kubectl apply -f k8s/services/processing-service.yaml
kubectl apply -f k8s/services/status-service.yaml
kubectl apply -f k8s/services/notification-service.yaml
kubectl apply -f k8s/services/api-gateway.yaml
kubectl apply -f k8s/services/frontend.yaml
```

## 🌐 4. Acessando a Aplicação

Como o Minikube roda isolado, você precisa liberar o acesso às portas.

### API Gateway (Backend)
Para que o Frontend consiga chamar a API em `localhost:8080`, você deve criar um túnel:

```powershell
kubectl port-forward service/api-gateway 8080:8080 -n g57
```
*Mantenha esse terminal aberto.*

### Frontend
Para acessar a interface visual no navegador:

```powershell
minikube service frontend -n g57
```
Isso abrirá automaticamente o navegador com a URL correta.

### Kubernetes Dashboard (Visualizar Logs e Pods)
```powershell
minikube dashboard
```

## ⚠️ Solução de Problemas Comuns

**Erro:** `Unable to connect to the server: dial tcp 127.0.0.1:xxxxx: connectex: No connection could be made...`
*   **Causa:** O Minikube parou ou o container Docker dele foi removido.
*   **Solução:** Rode `minikube start`.

**Erro:** `ImagePullBackOff` ou `ErrImagePull`
*   **Causa:** O Kubernetes não encontrou a imagem Docker.
*   **Solução:** Garanta que você buildou a imagem **dentro do ambiente do Minikube**:
    ```powershell
    # Apenas se usar Linux/Mac: eval $(minikube -p minikube docker-env)
    # No Windows, use o comando direto do minikube:
    minikube image build -t g57-video-service:latest ./services/video-service

    # Caso específico para o frontend (se o comando acima falhar):
    docker build -t g57-frontend:latest ./frontend
    minikube image load g57-frontend:latest
    ```

**Erro:** Conexão recusada no Frontend ao tentar logar
*   **Solução:** Verifique se o `port-forward` do API Gateway (passo 4) está rodando. O frontend tenta acessar `localhost:8080`.
