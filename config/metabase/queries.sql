# Queries SQL para Métricas no Metabase
# Cada seção é uma "Question" a ser criada no Metabase

# COMO USAR:
# 1. Acesse http://localhost:3001
# 2. Clique em "New Question" → "Native query"
# 3. Selecione a database correspondente
# 4. Cole a query, dê um nome e salve
# 5. Adicione as Questions ao Dashboard

# ══════════════════════════════════════════════════════════════════════════════
#  DATABASE: video_db
# ══════════════════════════════════════════════════════════════════════════════

-- [1] Total de Vídeos por Status (Pie Chart / Bar)
-- Question: "Status dos Vídeos"
SELECT
  status,
  COUNT(*)          AS total,
  ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER (), 1) AS percentual
FROM videos
GROUP BY status
ORDER BY total DESC;

-- [2] Uploads por Dia (últimos 30 dias) (Line Chart)
-- Question: "Uploads por Dia"
SELECT
  DATE(created_at)               AS dia,
  COUNT(*)                        AS uploads,
  COUNT(CASE WHEN status = 'completed' THEN 1 END) AS concluidos,
  COUNT(CASE WHEN status = 'failed'    THEN 1 END) AS falhas
FROM videos
WHERE created_at >= NOW() - INTERVAL '30 days'
GROUP BY DATE(created_at)
ORDER BY dia;

-- [3] Taxa de Sucesso de Processamento (Scalar / KPI)
-- Question: "Taxa de Sucesso (%)"
SELECT
  ROUND(
    COUNT(CASE WHEN status = 'completed' THEN 1 END) * 100.0 / NULLIF(COUNT(*), 0)
  , 1) AS taxa_sucesso_pct
FROM videos
WHERE status IN ('completed', 'failed');

-- [4] Tempo Médio de Processamento por Dia (Bar Chart)
-- Question: "Tempo Médio de Processamento (seg)"
SELECT
  DATE(processing_started_at)                                         AS dia,
  ROUND(AVG(EXTRACT(EPOCH FROM (processing_completed_at - processing_started_at))), 0) AS media_segundos,
  MAX(EXTRACT(EPOCH FROM (processing_completed_at - processing_started_at)))            AS maximo_segundos,
  MIN(EXTRACT(EPOCH FROM (processing_completed_at - processing_started_at)))            AS minimo_segundos
FROM videos
WHERE status = 'completed'
  AND processing_started_at IS NOT NULL
  AND processing_completed_at IS NOT NULL
GROUP BY DATE(processing_started_at)
ORDER BY dia;

-- [5] Top Usuários por Volume de Uploads (Table)
-- Question: "Top Usuários por Uploads"
SELECT
  user_id,
  COUNT(*)                                                              AS total_videos,
  COUNT(CASE WHEN status = 'completed' THEN 1 END)                     AS concluidos,
  COUNT(CASE WHEN status = 'failed'    THEN 1 END)                     AS falhas,
  ROUND(SUM(size_bytes) / 1024.0 / 1024.0 / 1024.0, 2)               AS storage_total_gb,
  ROUND(AVG(size_bytes) / 1024.0 / 1024.0, 1)                         AS tamanho_medio_mb
FROM videos
GROUP BY user_id
ORDER BY total_videos DESC
LIMIT 10;

-- [6] Distribuição de Tamanho dos Vídeos (Bar Chart)
-- Question: "Distribuição de Tamanho dos Vídeos"
SELECT
  CASE
    WHEN size_bytes < 10485760   THEN '< 10 MB'
    WHEN size_bytes < 52428800   THEN '10–50 MB'
    WHEN size_bytes < 104857600  THEN '50–100 MB'
    WHEN size_bytes < 524288000  THEN '100–500 MB'
    ELSE '> 500 MB'
  END AS faixa_tamanho,
  COUNT(*) AS quantidade
FROM videos
GROUP BY faixa_tamanho
ORDER BY MIN(size_bytes);

-- [7] Taxa de Compressão ZIP (Scatter / Bar)
-- Question: "Eficiência de Compressão (%)"
SELECT
  DATE(processing_completed_at)                                          AS dia,
  ROUND(AVG((size_bytes - zip_size_bytes) * 100.0 / NULLIF(size_bytes, 0)), 1) AS reducao_media_pct,
  ROUND(AVG(zip_size_bytes) / 1024.0 / 1024.0, 1)                      AS zip_medio_mb
FROM videos
WHERE status = 'completed'
  AND zip_size_bytes IS NOT NULL
  AND size_bytes > 0
GROUP BY DATE(processing_completed_at)
ORDER BY dia;

-- [8] Total de Frames Extraídos por Dia (Line Chart)
-- Question: "Frames Extraídos por Dia"
SELECT
  DATE(processing_completed_at) AS dia,
  SUM(frame_count)              AS frames_total,
  ROUND(AVG(frame_count), 0)    AS frames_media,
  COUNT(*)                      AS videos_processados
FROM videos
WHERE status = 'completed'
  AND frame_count IS NOT NULL
GROUP BY DATE(processing_completed_at)
ORDER BY dia;

-- [9] Vídeos com Reprocessamento (Table - alertas)
-- Question: "Vídeos com Retentativas"
SELECT
  id,
  user_id,
  original_name,
  status,
  retry_count,
  error_message,
  created_at
FROM videos
WHERE retry_count > 0
ORDER BY retry_count DESC, created_at DESC;

-- [10] Fila Atual (KPI Cards)
-- Question: "Fila de Processamento Atual"
SELECT
  COUNT(CASE WHEN status = 'pending'    THEN 1 END) AS pendentes,
  COUNT(CASE WHEN status = 'queued'     THEN 1 END) AS na_fila,
  COUNT(CASE WHEN status = 'processing' THEN 1 END) AS processando,
  COUNT(CASE WHEN status = 'completed'  THEN 1 END) AS concluidos_hoje,
  COUNT(CASE WHEN status = 'failed'     THEN 1 END) AS falhas_hoje
FROM videos
WHERE created_at >= CURRENT_DATE;

-- [11] Armazenamento Total Utilizado (Scalar)
-- Question: "Storage Total (GB)"
SELECT
  ROUND(SUM(size_bytes)     / 1024.0 / 1024.0 / 1024.0, 2) AS storage_videos_gb,
  ROUND(SUM(zip_size_bytes) / 1024.0 / 1024.0 / 1024.0, 2) AS storage_zips_gb,
  COUNT(*) AS total_videos
FROM videos
WHERE status = 'completed';

-- [12] Velocidade de Processamento (MB/s média) (Scalar)
-- Question: "Velocidade Média de Processamento"
SELECT
  ROUND(
    AVG(size_bytes / 1024.0 / 1024.0 /
        NULLIF(EXTRACT(EPOCH FROM (processing_completed_at - processing_started_at)), 0))
  , 2) AS velocidade_media_mb_s
FROM videos
WHERE status = 'completed'
  AND processing_started_at IS NOT NULL
  AND processing_completed_at IS NOT NULL;


# ══════════════════════════════════════════════════════════════════════════════
#  DATABASE: processing_db
# ══════════════════════════════════════════════════════════════════════════════

-- [13] Desempenho por Worker (Table / Bar)
-- Question: "Desempenho por Worker"
SELECT
  worker_id,
  COUNT(*)                                                  AS jobs_total,
  COUNT(CASE WHEN status = 'completed' THEN 1 END)          AS concluidos,
  COUNT(CASE WHEN status = 'failed'    THEN 1 END)          AS falhas,
  ROUND(AVG(duration_seconds), 0)                           AS duracao_media_seg,
  ROUND(COUNT(CASE WHEN status = 'completed' THEN 1 END) * 100.0 / NULLIF(COUNT(*), 0), 1) AS taxa_sucesso_pct
FROM processing_jobs
WHERE status IN ('completed', 'failed')
GROUP BY worker_id
ORDER BY jobs_total DESC;

-- [14] Erros de Processamento Mais Comuns (Table)
-- Question: "Erros Mais Comuns"
SELECT
  error_message,
  COUNT(*) AS ocorrencias,
  MAX(created_at) AS ultima_ocorrencia
FROM processing_jobs
WHERE status = 'failed'
  AND error_message IS NOT NULL
GROUP BY error_message
ORDER BY ocorrencias DESC;

-- [15] Jobs Ativos Agora (Scalar - Live)
-- Question: "Jobs Rodando Agora"
SELECT
  COUNT(CASE WHEN status = 'running'  THEN 1 END) AS rodando,
  COUNT(CASE WHEN status = 'pending'  THEN 1 END) AS aguardando,
  ROUND(AVG(EXTRACT(EPOCH FROM (NOW() - started_at))) / 60.0, 1) AS tempo_medio_rodando_min
FROM processing_jobs
WHERE status IN ('running', 'pending');

-- [16] Throughput Diário de Processamento (Bar)
-- Question: "Jobs Processados por Dia"
SELECT
  DATE(completed_at)           AS dia,
  COUNT(*)                      AS jobs_concluidos,
  ROUND(AVG(duration_seconds), 0) AS duracao_media_seg,
  SUM(duration_seconds)         AS duracao_total_seg
FROM processing_jobs
WHERE status = 'completed'
  AND completed_at IS NOT NULL
GROUP BY DATE(completed_at)
ORDER BY dia;


# ══════════════════════════════════════════════════════════════════════════════
#  DATABASE: notification_db
# ══════════════════════════════════════════════════════════════════════════════

-- [17] Notificações por Status (Pie)
-- Question: "Status das Notificações"
SELECT
  status,
  COUNT(*) AS total
FROM notifications
GROUP BY status;

-- [18] Taxa de Entrega de Emails (Scalar)
-- Question: "Taxa de Entrega de Emails (%)"
SELECT
  ROUND(
    COUNT(CASE WHEN status = 'sent' THEN 1 END) * 100.0 / NULLIF(COUNT(*), 0)
  , 1) AS taxa_entrega_pct,
  COUNT(CASE WHEN status = 'sent'   THEN 1 END) AS enviados,
  COUNT(CASE WHEN status = 'failed' THEN 1 END) AS falhas,
  COUNT(CASE WHEN status = 'pending' THEN 1 END) AS pendentes
FROM notifications;

-- [19] Notificações de Erro vs Sucesso por Dia (Stacked Bar)
-- Question: "Notificações por Tipo por Dia"
SELECT
  DATE(created_at)                                                       AS dia,
  COUNT(CASE WHEN subject LIKE '%concluído%' THEN 1 END)                AS sucesso,
  COUNT(CASE WHEN subject LIKE '%Falha%' OR subject LIKE '%falha%' THEN 1 END) AS falha
FROM notifications
GROUP BY DATE(created_at)
ORDER BY dia;


# ══════════════════════════════════════════════════════════════════════════════
#  DATABASE: auth_db
# ══════════════════════════════════════════════════════════════════════════════

-- [20] Total de Usuários Ativos (Scalar)
-- Question: "Usuários Ativos"
SELECT
  COUNT(*)                                          AS total_usuarios,
  COUNT(CASE WHEN is_active = true  THEN 1 END)    AS ativos,
  COUNT(CASE WHEN is_active = false THEN 1 END)    AS inativos,
  COUNT(CASE WHEN role = 'admin'    THEN 1 END)    AS admins,
  COUNT(CASE WHEN email_verified = true THEN 1 END) AS verificados
FROM users;

-- [21] Novos Usuários por Semana (Bar)
-- Question: "Novos Usuários por Semana"
SELECT
  DATE_TRUNC('week', created_at) AS semana,
  COUNT(*)                        AS novos_usuarios
FROM users
GROUP BY DATE_TRUNC('week', created_at)
ORDER BY semana;

-- [22] Logins por Dia (Line)
-- Question: "Logins por Dia"
SELECT
  DATE(created_at)  AS dia,
  COUNT(*)           AS logins
FROM audit_logs
WHERE action = 'login'
GROUP BY DATE(created_at)
ORDER BY dia;

-- [23] Usuários sem Login Recente (30+ dias) (Table)
-- Question: "Usuários Inativos (30+ dias)"
SELECT
  email,
  name,
  last_login_at,
  created_at,
  EXTRACT(DAY FROM NOW() - last_login_at) AS dias_sem_login
FROM users
WHERE is_active = true
  AND (last_login_at IS NULL OR last_login_at < NOW() - INTERVAL '30 days')
ORDER BY last_login_at ASC NULLS FIRST;
