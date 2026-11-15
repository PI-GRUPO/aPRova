import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;

class ToneladasParaQuilogramasTest {
    @Test
    void converteUmaUnidade() {
        assertEquals(1000, ToneladasParaQuilogramas.converter(1), 1e-9);
    }
}
