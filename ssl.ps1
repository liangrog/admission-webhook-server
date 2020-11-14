#call example .\ssl.ps1 -CommonName admission-webhook.tools.svc
param ([Parameter(Mandatory)] [string] $CommonName, [int] $ExpirationDays=3650)
$secretDir="chart/ssl"
mkdir -Force -Path $secretDir
Set-Location $secretDir
Remove-Item * -Recurse
# Generate the CA cert and private key
openssl req -nodes -new -x509 -days $ExpirationDays -keyout ca.key -out ca.crt -subj "/CN=Admission Controller Webhook Server CA"
Get-Content ca.key > server.pem
Get-Content ca.crt >> server.pem
# Generate the private key for the webhook server
openssl genrsa -out tls.key 2048
# Generate a Certificate Signing Request (CSR) for the private key, and sign it with the private key of the CA.
openssl req -new -days $ExpirationDays -key tls.key -subj "/CN=$commonName" `
    | openssl x509 -days $ExpirationDays -req -CA ca.crt -CAkey ca.key -CAcreateserial -out tls.crt
Set-Location ..\..

