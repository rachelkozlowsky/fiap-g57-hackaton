## 🏗️ Arquitetura

### Microsserviços com Database per Service

> 💡 **Dica:** Você pode abrir o arquivo `arquitetura-hakaton.drawio.xml` no [draw.io](https://app.diagrams.net/) para visualizar o desenho completo.


```
┌─────────────────────────────────────────────────────────────────┐
│                          API Gateway                             │
│                    (Autenticação & Roteamento)                   │
└──────────────────────┬──────────────────────────────────────────┘
                       │
        ┌──────────────┼──────────────┬──────────────┐
        │              │               │              │
   ┌────▼────┐   ┌────▼────┐    ┌────▼────┐   ┌─────▼─────┐
   │  Auth   │   │  Video  │    │ Status  │   │Notification│
   │ Service │   │ Service │    │ Service │   │  Service   │
   └────┬────┘   └────┬────┘    └────┬────┘   └─────▲─────┘
        │             │              │              │
   ┌────▼────┐   ┌───▼─────┐   ┌────▼────┐         │
   │ Auth DB │   │Video DB │   │Status DB│         │
   └─────────┘   └────┬────┘   └─────────┘         │
                      │                             │
                      │         ┌────────┐          │
                      └────────►│RabbitMQ│──────────┘
                                └────┬───┘
                                     │
                              ┌──────▼──────┐
                              │ Processing  │
                              │   Service   │
                              └──────┬──────┘
                                     │
                              ┌──────▼──────┐
                              │Processing DB│
                              └─────────────┘
```

### Componentes

1. **API Gateway** (Node.js/Express)
   - Autenticação JWT
   - Rate limiting
   - Roteamento de requisições

2. **Auth Service** (Go)
   - Registro e login de usuários
   - Geração de tokens JWT
   - Gerenciamento de sessões
   - **Database**: `auth_db` (PostgreSQL)
     - Tabelas: users, sessions, audit_logs

3. **Video Service** (Go)
   - Upload de vídeos
   - Validação de formatos
   - Publicação na fila
   - **Database**: `video_db` (PostgreSQL)
     - Tabelas: videos
   - **Comunicação**: HTTP com Auth Service

4. **Processing Service** (Go)
   - Consumo de mensagens
   - Extração de frames com FFmpeg
   - Geração de ZIP
   - **Database**: `processing_db` (PostgreSQL)
     - Tabelas: processing_jobs, system_metrics
   - **Comunicação**: HTTP com Video Service

5. **Status Service** (Go)
   - Consulta de status de processamento
   - Listagem de vídeos do usuário
   - Cache com Redis
   - **Database**: `status_db` (PostgreSQL)
     - Tabelas: status_cache, query_logs
   - **Comunicação**: HTTP com Video Service e Auth Service

6. **Notification Service** (Go)
   - Envio de emails
   - Notificações de conclusão/erro
   - **Database**: `notification_db` (PostgreSQL)
     - Tabelas: notifications, notification_templates
   - **Comunicação**: HTTP com Auth Service e Video Service

### Padrões de Arquitetura Implementados

- ✅ **Database per Service**: Cada microserviço tem seu próprio banco de dados
- ✅ **API Gateway Pattern**: Ponto único de entrada
- ✅ **Event-Driven Architecture**: Comunicação assíncrona via RabbitMQ
- ✅ **CQRS**: Separação de leitura (Status Service) e escrita (Video Service)
- ✅ **Cache-Aside Pattern**: Redis para otimização de consultas

## 🚀 Funcionalidades

### ✅ Requisitos Funcionais Implementados

- [x] Processamento paralelo de múltiplos vídeos
- [x] Fila de mensagens para evitar perda de requisições
- [x] Autenticação com usuário e senha
- [x] Listagem de status dos vídeos por usuário
- [x] Notificação por email em caso de erro/sucesso

### ✅ Requisitos Técnicos Implementados

- [x] Persistência de dados (PostgreSQL - Database per Service)
- [x] Arquitetura escalável (Kubernetes)
- [x] Versionamento no GitHub
- [x] Testes automatizados
- [x] CI/CD com GitHub Actions
- [x] Containerização com Docker
- [x] Mensageria com RabbitMQ
- [x] Cache com Redis

## 📦 Stack Tecnológica

- **Backend**: Go 1.21
- **API Gateway**: Node.js 20
- **Bancos de Dados**: 
  - PostgreSQL 15 (5 instâncias - uma por serviço)
- **Cache**: Redis 7
- **Mensageria**: RabbitMQ 3.12
- **Storage**: MinIO (S3-compatible)
- **Containerização**: Docker
- **Orquestração**: Kubernetes
- **Monitoramento**: Prometheus + Grafana
- **CI/CD**: GitHub Actions


