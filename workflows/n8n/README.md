# Workflows do n8n

Esta pasta receberá os workflows exportados como JSON, revisados para serem
compartilhados e importados nas instalações locais ou no futuro servidor.
Ainda não há workflows exportados aqui.

`../exemplos/` contém exemplos de eventos do aPRova. Esses arquivos não são
workflows e não devem ser importados no editor do n8n.

## Executar em uma máquina nova

Na raiz do projeto, com Docker e Docker Compose instalados:

```sh
mkdir -p .n8n
docker compose up -d n8n-aprova
```

Abra <http://localhost:5978/> e crie sua própria conta na primeira execução.
Cada máquina mantém seus dados em uma pasta `.n8n/` independente, ignorada
pelo Git. Em Linux, essa pasta precisa permitir escrita pelo usuário `node`
do container (UID/GID `1000:1000`). Se ocorrer `EACCES`, ajuste a propriedade
da pasta para esse usuário.

O `.env` é opcional para abrir o editor com os valores padrão. Para
personalizar porta, URL e fuso, use as variáveis documentadas no
[README principal](../../README.md#n8n-opcional).

## Exportar para o repositório

1. Abra o workflow no seu n8n e use o menu do workflow para baixar o JSON
   (opção **Download**).
2. Revise uma cópia do JSON antes de colocá-la nesta pasta. Remova tokens,
   senhas, headers de autenticação, segredos usados em nós Code, dados
   pessoais e dados de teste fixados nos nós (`pinData`). Revise também
   nomes e referências das credenciais pessoais.
3. Substitua valores específicos da sua instalação por exemplos e documente
   as credenciais, URLs e configurações necessárias para aquele workflow.
4. Salve com um nome descritivo, como `notificacao-slack.json`, e inclua o
   arquivo no commit.

O JSON exportado pode conter nomes e IDs de credenciais e valores inseridos
diretamente nos nós. A revisão precisa acontecer mesmo sem exportar o banco
de credenciais. Veja a
[documentação de exportação do n8n](https://docs.n8n.io/workflows/export-import/).

Não inclua nesta pasta o banco SQLite, o arquivo `config`, exports de
credenciais ou cópias da pasta `.n8n/`.

## Importar e configurar

1. Abra sua própria instância do n8n.
2. Use **Import from File** no menu do workflow e selecione o JSON desta
   pasta.
3. Configure as credenciais da sua instalação e associe-as aos nós.
4. Ajuste os endereços e teste antes de publicar ou ativar o workflow,
   conforme a versão do n8n.

Para chamar a API do aPRova em execução na máquina host, use
`http://host.docker.internal:8080` nos nós HTTP (ajustando a porta da API se
necessário). A API precisa estar em execução com suas dependências.

## Levar ao servidor

No servidor, importe os JSONs revisados, configure as credenciais e os
segredos daquele ambiente e ajuste as URLs para os serviços acessíveis de
lá. Depois de testar, publique ou ative os workflows.

A conta, as credenciais e os dados da sua instalação local permanecem na
sua máquina. Não copie o banco pessoal para distribuir os workflows.

O Compose do repositório é para desenvolvimento local. A publicação em um
servidor exige configuração própria de acesso, persistência e credenciais.
A esteira atual não importa workflows nem faz deploy do n8n.
