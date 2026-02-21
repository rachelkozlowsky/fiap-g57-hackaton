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

**Importante:** As migrações de banco de dados são executadas automaticamente pelos próprios serviços ao iniciarem, usando `golang-migrate`. Não é necessário executar scripts SQL manualmente.

## 🧹 Limpeza Completa (Reiniciar do Zero)

Se você quiser deletar tudo e começar do zero:

```powershell
.\cleanup-k8s.ps1
```

Este script irá:
- Deletar o namespace `g57` (removendo todos os recursos)
- Opcionalmente, limpar as imagens Docker do Minikube
- Deixar o ambiente pronto para um novo deploy

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

Como o Minikube roda isolado, você precisa liberar o acesso às portas. Para garantir que tudo funcione em `localhost` como esperado, execute os seguintes comandos em terminais separados:

### 🖥️ Frontend & API (Essencial)

**Frontend:** (Acesse em http://localhost:5173)
```powershell
kubectl port-forward service/frontend 5173:5173 -n g57
```

**API Gateway:** (Backend em http://localhost:8080)
```powershell
kubectl port-forward service/api-gateway 8080:8080 -n g57
```

### 📊 Monitoramento & Infraestrutura (Opcional)

**Grafana:** (Acesse em http://localhost:3000 - admin/admin)
```powershell
kubectl port-forward service/grafana 3000:3000 -n g57
```

**Prometheus:** (Acesse em http://localhost:9090)
```powershell
kubectl port-forward service/prometheus 9090:9090 -n g57
```

**RabbitMQ Management:** (Acesse em http://localhost:15672 - g57/g57123456)
```powershell
kubectl port-forward service/rabbitmq 15672:15672 -n g57
```

**MinIO Console:** (Acesse em http://localhost:9001 - g57/g57123456)
```powershell
kubectl port-forward service/minio 9001:9001 -n g57
```

---
## 📈 Como Acompanhar o Teste de Carga

Para validar o escalonamento horizontal (HPA) e a performance da aplicação:

1.  **Inicie o Teste de Carga (K6):**
    O teste é executado como um Job dentro do cluster para minimizar a latência de rede.
    ```powershell
    # Se o job já existiu, delete primeiro
    kubectl delete job k6-load-test -n g57
    
    # Inicie o teste
    kubectl apply -f k8s/infra/k6-job.yaml
    ```

2.  **Monitore o Escalonamento em Tempo Real:**
    Abra um terminal e observe o número de réplicas aumentando automaticamente:
    ```powershell
    kubectl get hpa -n g57 -w
    ```

3.  **Visualize no Grafana:**
    Acesse o dashboard "G57 Microservices" (http://localhost:3000) e observe:
    *   **Pod Count (Scaling):** Novo painel mostrando a quantidade de pods de cada serviço subindo.
    *   **HTTP Request Rate:** O aumento no volume de requisições.
    *   **CPU/Memory:** O consumo de recursos disparando.


### Alternativa: Minikube Service (Sem Port-Forward fixo)
Se não quiser prender terminais, você pode pedir ao minikube para abrir cada serviço diretamente:

```powershell
minikube service frontend -n g57
minikube service grafana -n g57
```

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
