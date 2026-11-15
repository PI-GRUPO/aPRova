import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;

class PolegadasParaMilimetrosTest {
    @Test
    void converteUmaUnidade() {
        assertEquals(25.4, PolegadasParaMilimetros.converter(1), 1e-9);
    }
}
