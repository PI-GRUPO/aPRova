import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;

class MetrosParaCentimetrosTest {
    @Test
    void converteUmaUnidade() {
        assertEquals(100, MetrosParaCentimetros.converter(1), 1e-9);
    }
}
