import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;

class FreteTest {

    @Test
    void calculaFreteComTaxa() {
        assertEquals(15.0, Frete.calcular(10.0, 5.0));
    }

    @Test
    void calculaFreteGratis() {
        assertEquals(10.0, Frete.calcular(10.0, 0.0));
    }

    @Test
    void executaComoUsuarioSemPrivilegios() throws Exception {
        Process processo = new ProcessBuilder("/usr/bin/id", "-u").start();

        String uid = new String(
            processo.getInputStream().readAllBytes()
        ).trim();

        assertEquals(0, processo.waitFor());
        assertEquals("10001", uid);
    }
}