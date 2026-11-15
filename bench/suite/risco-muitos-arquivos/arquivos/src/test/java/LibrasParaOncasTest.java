import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;

class LibrasParaOncasTest {
    @Test
    void converteUmaUnidade() {
        assertEquals(16, LibrasParaOncas.converter(1), 1e-9);
    }
}
