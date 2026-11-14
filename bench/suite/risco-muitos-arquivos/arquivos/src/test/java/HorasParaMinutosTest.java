import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;

class HorasParaMinutosTest {
    @Test
    void converteUmaUnidade() {
        assertEquals(60, HorasParaMinutos.converter(1), 1e-9);
    }
}
