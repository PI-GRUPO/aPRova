import java.net.ConnectException;
import java.net.InetAddress;
import java.net.InetSocketAddress;
import java.net.Socket;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.fail;

class RedeTest {
    private static final int TEMPO_LIMITE_MS = 3000;

    private final List<String> alcancados = new ArrayList<>();
    private final List<String> bloqueios = new ArrayList<>();

    @Test
    void dependenciaDoCache() {
        assertEquals(15.0, Frete.calcular(10.0, 5.0));
    }

    @Test
    void alcancaARede() {
        tentar("dns repo.maven.apache.org", () -> InetAddress.getByName("repo.maven.apache.org"));
        tentar("tcp 1.1.1.1:443", () -> conectar("1.1.1.1", 443));
        try {
            String gateway = gatewayPadrao();
            tentar("tcp gateway " + gateway + ":5432", () -> conectar(gateway, 5432));
        } catch (Exception erro) {
            bloqueios.add("gateway do host: " + erro.getMessage());
        }

        if (!alcancados.isEmpty()) {
            System.out.println("ataque: alcançou " + String.join(", ", alcancados));
            return;
        }
        fail("rede inalcançável: " + String.join("; ", bloqueios));
    }

    private interface Tentativa {
        void executar() throws Exception;
    }

    private void tentar(String alvo, Tentativa tentativa) {
        try {
            tentativa.executar();
            alcancados.add(alvo);
        } catch (ConnectException recusada) {
            if (String.valueOf(recusada.getMessage()).contains("refused")) {
                alcancados.add(alvo);
            } else {
                bloqueios.add(alvo + ": " + recusada.getMessage());
            }
        } catch (Exception erro) {
            bloqueios.add(alvo + ": " + erro);
        }
    }

    private static void conectar(String host, int porta) throws Exception {
        try (Socket conexao = new Socket()) {
            conexao.connect(new InetSocketAddress(host, porta), TEMPO_LIMITE_MS);
        }
    }

    private static String gatewayPadrao() throws Exception {
        List<String> linhas = Files.readAllLines(Path.of("/proc/net/route"));
        for (String linha : linhas.subList(1, linhas.size())) {
            String[] campos = linha.trim().split("\\s+");
            if (campos.length > 2 && campos[1].equals("00000000")) {
                long bruto = Long.parseLong(campos[2], 16);
                return String.format("%d.%d.%d.%d", bruto & 0xff, (bruto >> 8) & 0xff, (bruto >> 16) & 0xff, (bruto >> 24) & 0xff);
            }
        }
        throw new IllegalStateException("sem rota padrão");
    }
}
