import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;

class DiasParaHorasTest {
    @Test
    void converteUmaUnidade() {
        assertEquals(24, DiasParaHoras.converter(1), 1e-9);
    }
}
