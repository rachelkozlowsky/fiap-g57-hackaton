Write-Host "Limpando ambiente Kubernetes..." -ForegroundColor Yellow

Write-Host "Deletando PVCs (volumes persistentes)..."
kubectl delete pvc --all -n g57 --ignore-not-found=true

Write-Host "Deletando namespace g57..."
kubectl delete namespace g57 --ignore-not-found=true

Write-Host "Aguardando exclusão completa do namespace..."
Start-Sleep -Seconds 10

Write-Host "Deseja limpar as imagens Docker do Minikube? (S/N)"
$response = Read-Host
if ($response -eq 'S' -or $response -eq 's') {
    Write-Host "Limpando imagens antigas..."
    minikube image rm g57-auth-service:latest
    minikube image rm g57-video-service:latest
    minikube image rm g57-processing-service:latest
    minikube image rm g57-notification-service:latest
    minikube image rm g57-status-service:latest
    minikube image rm g57-api-gateway:latest
    minikube image rm g57-frontend:latest
}

Write-Host "Limpeza concluída!" -ForegroundColor Green
Write-Host ""
Write-Host "Para reiniciar do zero, execute:" -ForegroundColor Cyan
Write-Host "  .\deploy-k8s.ps1" -ForegroundColor White
