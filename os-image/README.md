# Imagem Appliance do Meu NAS

## Objetivo

Criar um sistema operacional instalável para Raspberry Pi 4 e 5, baseado em Ubuntu Server ARM64, em que a pessoa grava a imagem no SD/SSD, liga o Pi e acessa o painel web do Meu NAS.

## Decisão técnica principal

Não vamos "compilar um Ubuntu do zero" no começo. Isso aumenta muito o custo de manutenção e segurança.

A abordagem recomendada é:

1. usar uma imagem oficial do Ubuntu Server ARM64 para Raspberry Pi como base;
2. injetar uma configuração de primeiro boot com cloud-init;
3. instalar o Meu NAS em `/opt/meu-nas`;
4. registrar serviços systemd;
5. configurar nginx;
6. deixar atualizações de segurança do Ubuntu funcionando.

Isso gera uma experiência de produto pronto sem assumir a manutenção de uma distribuição Linux inteira.

## Fluxo de build

```text
Ubuntu Server ARM64 image
        |
        v
customize-image.sh
        |
        +-- injeta cloud-init
        +-- copia scripts de primeiro boot
        +-- copia pacote do app
        |
        v
meu-nas-ubuntu-rpi-arm64.img.xz
```

## Estrutura proposta

```text
os-image/
├── README.md
├── cloud-init/
│   ├── meta-data
│   ├── network-config
│   └── user-data
├── first-boot/
│   ├── meu-nas-first-boot.service
│   └── meu-nas-first-boot.sh
├── manifests/
│   └── packages.apt
└── scripts/
    ├── build-app-bundle.sh
    └── customize-image.sh
```

## Requisitos do host de build

Use Linux para gerar a imagem final. Pode ser Ubuntu em máquina física, VM ou WSL2 com suporte a loop devices configurado.

No Windows, o caminho recomendado é uma VM Ubuntu no Hyper-V. Veja `os-image/hyperv/README.md`.

Pacotes úteis:

```bash
sudo apt-get update
sudo apt-get install -y \
  qemu-user-static binfmt-support xz-utils kpartx rsync cloud-image-utils \
  dosfstools e2fsprogs parted
```

## Como usar

1. Baixe a imagem Ubuntu Server ARM64 para Raspberry Pi.
2. Compile o bundle do app:

```bash
os-image/scripts/build-app-bundle.sh
```

3. Customize a imagem. Esta etapa monta a imagem, instala os pacotes apt, cria o virtualenv Python dentro do rootfs ARM64 e habilita os serviços:

```bash
sudo os-image/scripts/customize-image.sh \
  ubuntu-server-rpi-arm64.img.xz \
  dist/meu-nas-app.tar.gz \
  dist/meu-nas-ubuntu-rpi-arm64.img
```

4. Comprima a imagem final:

```bash
xz -T0 -9 dist/meu-nas-ubuntu-rpi-arm64.img
```

5. Grave com Raspberry Pi Imager, Balena Etcher ou `dd`.

## Primeiro acesso

Depois do boot, acesse:

```text
http://meu-nas.local
```

ou o IP mostrado pelo roteador.

Credenciais iniciais planejadas:

```text
usuário: admin
senha: alterada-no-primeiro-acesso
```

A senha inicial é gerada no primeiro boot e gravada em:

```text
/etc/meu-nas-initial-password
```

No produto final, a UI deve obrigar troca de senha no primeiro acesso e apagar esse arquivo.

## Armadilhas comuns

- Imagem pronta não deve embutir chave JWT fixa. O primeiro boot deve gerar `MEUNAS_SECRET_KEY`.
- Não use senha padrão fixa em produção.
- O backend não deve rodar como root.
- A imagem precisa tolerar boot sem internet; por isso dependências apt e Python são instaladas durante o build.
- Expansão de filesystem deve acontecer antes de criar dados persistentes grandes.
- Raspberry Pi 4 e 5 podem exigir firmware/kernel diferentes; por isso é melhor herdar a imagem oficial do Ubuntu para Pi.

## Próximos passos

1. Transformar o backend em pacote instalável offline.
2. Gerar frontend estático no bundle.
3. Criar wizard de primeiro acesso para trocar senha.
4. Adicionar mDNS com `avahi-daemon` para `meu-nas.local`.
5. Criar pipeline de release que gera `.img.xz` e checksum SHA256.
