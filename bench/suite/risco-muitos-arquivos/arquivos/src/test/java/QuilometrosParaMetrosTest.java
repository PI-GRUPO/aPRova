import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;

class QuilometrosParaMetrosTest {
    @Test
    void converteUmaUnidade() {
        assertEquals(1000, QuilometrosParaMetros.converter(1), 1e-9);
    }
}
