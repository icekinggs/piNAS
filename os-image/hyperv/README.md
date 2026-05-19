# Build VM no Hyper-V

Use esta VM Ubuntu como host de build para gerar a imagem `.img.xz` do Meu NAS para Raspberry Pi.

## Por que Hyper-V

O build da imagem precisa de recursos Linux que são chatos no Windows puro:

- loop devices;
- mount de partições dentro de `.img`;
- chroot ARM64 com `qemu-aarch64-static`;
- `rsync`, `xz`, `parted`, `e2fsprogs`;
- permissões de root reais.

Hyper-V resolve isso com uma VM Ubuntu x86_64 rodando as ferramentas de build.

## Requisitos no Windows

1. Windows Pro, Enterprise ou Education com Hyper-V habilitado.
2. ISO do Ubuntu Server x86_64 para instalar a VM de build.
3. Pelo menos 4 vCPUs, 8 GB RAM e 80 GB de disco para trabalhar confortável.

Habilitar Hyper-V, se necessário:

```powershell
Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V -All
```

Reinicie o Windows depois.

## Criar a VM

Abra PowerShell como administrador:

```powershell
.\os-image\hyperv\create-build-vm.ps1 `
  -VmName "MeuNAS-Build" `
  -IsoPath "C:\ISO\ubuntu-server.iso" `
  -VmRoot "C:\HyperV" `
  -MemoryGB 8 `
  -CpuCount 4 `
  -DiskGB 100
```

Depois instale o Ubuntu normalmente pela tela da VM.

## Preparar Ubuntu dentro da VM

Depois de instalar e logar na VM:

```bash
sudo apt-get update
sudo apt-get install -y \
  git curl ca-certificates build-essential \
  qemu-user-static binfmt-support xz-utils kpartx rsync \
  cloud-image-utils dosfstools e2fsprogs parted \
  nodejs npm python3 python3-venv python3-pip
```

Clone ou copie este projeto para a VM:

```bash
git clone <repo-do-projeto> meu-nas
cd meu-nas
```

Se ainda não tiver repositório Git remoto, copie a pasta via compartilhamento de rede, `scp` ou Enhanced Session.

## Gerar a imagem

Baixe a imagem oficial do Ubuntu Server ARM64 para Raspberry Pi e coloque na pasta `dist/`.

```bash
mkdir -p dist
# exemplo: dist/ubuntu-server-rpi-arm64.img.xz
```

Compile o bundle:

```bash
os-image/scripts/build-app-bundle.sh
```

Customize a imagem:

```bash
sudo os-image/scripts/customize-image.sh \
  dist/ubuntu-server-rpi-arm64.img.xz \
  dist/meu-nas-app.tar.gz \
  dist/meu-nas-ubuntu-rpi-arm64.img
```

Comprima:

```bash
xz -T0 -9 dist/meu-nas-ubuntu-rpi-arm64.img
sha256sum dist/meu-nas-ubuntu-rpi-arm64.img.xz > dist/meu-nas-ubuntu-rpi-arm64.img.xz.sha256
```

## Observações importantes

- A VM é x86_64, mas o target é ARM64. O `qemu-user-static` permite executar comandos dentro do rootfs ARM64.
- A imagem final deve ser testada em Raspberry Pi real; Hyper-V não emula boot de Raspberry Pi.
- Para teste sem hardware, o máximo razoável é validar montagem, serviços instalados e scripts. Boot completo do Pi precisa do firmware/boot chain do Raspberry Pi.
