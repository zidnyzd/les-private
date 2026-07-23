# Deployment

## CI/CD Pipeline

Menganut sistem **Proprietary**. Artinya, kode sumber Go tidak pernah dinaikkan ke GitHub. Proses *Build* dilakukan secara mandiri di mesin lokal/VPS Anda, lalu binary yang sudah jadi disuntikkan ke GitHub. GitHub hanya bertugas mengkopi-tempel berkas (*SCP*) kembali ke server produksi melalui GitHub Actions.

Alur:
```text
Server/Lokal: edit .go → go build → git add les-app templates/ scripts/ → git push
    ↓
GitHub Actions (deploy.yml):
    1. Checkout
    2. appleboy/ssh-action → systemctl stop
    3. appleboy/scp-action → SCP les-app + templates/ + scripts/
    4. appleboy/ssh-action → systemctl restart
```

## Env / Secrets (GitHub Repo)

Agar GitHub Actions bisa mengirim berkas ke VPS, Anda harus memasukkan Rahasia (Secrets) berikut di pengaturan repositori:
| Secret | Source |
|--------|--------|
| `VPS_HOST` | Sumopod public IP |
| `VPS_USER` | SSH username (`root`) |
| `VPS_SSH_KEY` | Private key for SSH authentication |

## Env / VPS (Systemd)

Di setel pada file berkas servis (`/etc/systemd/system/les-private.service`):


```ini
[Service]
ExecStart=/root/les-private/les-app
WorkingDirectory=/root/les-private
Environment=ADMIN_USERNAME=guru
Environment=ADMIN_PASSWORD=rahasia
Environment=ADMIN_DISPLAY="Guru Utama"
Restart=always
```


## Deploy Commands (Buku Panduan)

Cara merilis pembaruan jika Anda mengedit kode di server ini:
```bash
cd /root/les-private
go build -o les-app .
git add -f les-app templates/ scripts/
git commit -m "update: deskripsi pembaruan Anda"
git push origin main
```
> **Penting:** Parameter `-f` pada perintah *git add* diwajibkan karena file *binary* (`les-app`) sejatinya masuk dalam daftar blokir `.gitignore`.
