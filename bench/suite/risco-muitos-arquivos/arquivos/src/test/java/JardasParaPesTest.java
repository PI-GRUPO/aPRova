import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;

class JardasParaPesTest {
    @Test
    void converteUmaUnidade() {
        assertEquals(3, JardasParaPes.converter(1), 1e-9);
    }
}
