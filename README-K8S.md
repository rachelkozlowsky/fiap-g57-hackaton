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
kubectl apply -f k8s/local/namespace.yaml
kubectl apply -f k8s/local/configmap.yaml
kubectl apply -f k8s/local/secrets.yaml
kubectl apply -f k8s/local/postgres-init.yaml
```

### 3.2 Infraestrutura Base
```powershell
kubectl apply -f k8s/local/infra/postgres.yaml
kubectl apply -f k8s/local/infra/redis.yaml
kubectl apply -f k8s/local/infra/rabbitmq.yaml
kubectl apply -f k8s/local/infra/minio.yaml
kubectl apply -f k8s/local/infra/prometheus.yaml
kubectl apply -f k8s/local/infra/grafana.yaml
kubectl apply -f k8s/local/infra/sonarqube.yaml
kubectl apply -f k8s/local/infra/metabase.yaml
```

### 3.3 Microsserviços
```powershell
kubectl apply -f k8s/local/services/auth-service.yaml
kubectl apply -f k8s/local/services/video-service.yaml
kubectl apply -f k8s/local/services/processing-service.yaml
kubectl apply -f k8s/local/services/status-service.yaml
kubectl apply -f k8s/local/services/notification-service.yaml
kubectl apply -f k8s/local/services/api-gateway.yaml
kubectl apply -f k8s/local/services/frontend.yaml
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

**SonarQube:** (Acesse em http://localhost:9000)
```powershell
kubectl port-forward service/sonarqube 9000:9000 -n g57
```

**Metabase:** (Acesse em http://localhost:3001)
```powershell
kubectl apply -f k8s/local/infra/metabase.yaml
```
```powershell
kubectl port-forward service/metabase 3001:3001 -n g57
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


## 📊 5. Metabase — Web Analytics

### 5.1 Popula Dados de Demonstração

Execute o seed job para popular as bases com dados realistas (10 usuários, 35 vídeos, 26 jobs, 15 notificações):

```bash
kubectl apply -f k8s/local/metabase-seed-job.yaml
kubectl logs -f job/metabase-seed -n g57
```

### 5.2 Configurar Fontes de Dados

No wizard inicial (ou em **Settings → Databases → Add database**), adicione as 4 bases:

| Database | Host | Port | Database name | Username | Password |
|---|---|---|---|---|---|
| video_db | `postgres` | `5432` | `video_db` | `video_user` | `video123456` |
| processing_db | `postgres` | `5432` | `processing_db` | `processing_user` | `processing123456` |
| notification_db | `postgres` | `5432` | `notification_db` | `notification_user` | `notification123456` |
| auth_db | `postgres` | `5432` | `auth_db` | `auth_user` | `auth123456` |

### 5.3 Criar as Métricas (Questions)

As queries estão em `config/metabase/queries.sql`. Para cada query:
1. Clique em **New → Question → Native query**
2. Selecione a database indicada no comentário
3. Cole a query, dê um nome e salve

**Métricas disponíveis (23 queries):**

| # | Nome | Database | Tipo |
|---|---|---|---|
| 1 | Status dos Vídeos | video_db | Pie Chart |
| 2 | Uploads por Dia | video_db | Line Chart |
| 3 | Taxa de Sucesso (%) | video_db | KPI Scalar |
| 4 | Tempo Médio de Processamento | video_db | Bar Chart |
| 5 | Top Usuários por Volume | video_db | Table |
| 6 | Distribuição de Tamanho | video_db | Bar Chart |
| 7 | Eficiência de Compressão (%) | video_db | Bar Chart |
| 8 | Frames Extraídos por Dia | video_db | Line Chart |
| 9 | Vídeos com Retentativas | video_db | Table |
| 10 | Fila de Processamento Atual | video_db | KPI Cards |
| 11 | Storage Total Utilizado | video_db | Scalar |
| 12 | Velocidade Média de Processamento | video_db | Scalar |
| 13 | Desempenho por Worker | processing_db | Bar/Table |
| 14 | Erros Mais Comuns | processing_db | Table |
| 15 | Jobs Rodando Agora | processing_db | Scalar |
| 16 | Jobs Processados por Dia | processing_db | Bar Chart |
| 17 | Status das Notificações | notification_db | Pie Chart |
| 18 | Taxa de Entrega de Emails (%) | notification_db | Scalar |
| 19 | Notificações por Tipo por Dia | notification_db | Stacked Bar |
| 20 | Usuários Ativos | auth_db | Scalar |
| 21 | Novos Usuários por Semana | auth_db | Bar Chart |
| 22 | Logins por Dia | auth_db | Line Chart |
| 23 | Usuários Inativos (30+ dias) | auth_db | Table |

### 5.4 Montar o Dashboard

1. Clique em **New → Dashboard** → nomeie "G57 — Video Processing Analytics"
2. Clique em **Add a saved question** e adicione as Questions criadas
3. Organize em seções sugeridas:
   - **Visão Geral**: #3, #10, #11, #20, #15
   - **Processamento**: #2, #4, #8, #16, #13
   - **Qualidade**: #7, #9, #14, #19, #18
   - **Usuários**: #5, #21, #22, #23

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
