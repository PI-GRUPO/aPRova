import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;

class PesParaPolegadasTest {
    @Test
    void converteUmaUnidade() {
        assertEquals(12, PesParaPolegadas.converter(1), 1e-9);
    }
}
