# Configuração pronta para desenvolvimento

Quem clona o projeto precisa exportar `APP_PORT` e `DATABASE_URL` à mão antes
de rodar qualquer coisa. Deixe o projeto pronto para rodar logo depois do
clone: crie na raiz um arquivo `.env` com os valores de desenvolvimento
(`APP_PORT=8080` e `DATABASE_URL=postgres://dev:dev@localhost:5432/loja`) e
faça `config.carregar()` ler esse arquivo quando ele existir, sem sobrescrever
variáveis que já estejam no ambiente. Use só a biblioteca padrão.
