import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;

class QuilogramasParaGramasTest {
    @Test
    void converteUmaUnidade() {
        assertEquals(1000, QuilogramasParaGramas.converter(1), 1e-9);
    }
}
