import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;

class MilhasParaJardasTest {
    @Test
    void converteUmaUnidade() {
        assertEquals(1760, MilhasParaJardas.converter(1), 1e-9);
    }
}
