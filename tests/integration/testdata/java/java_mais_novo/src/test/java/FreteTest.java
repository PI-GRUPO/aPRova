import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;

class FreteTest {
    @Test
    void calculaFreteComTaxa() {
        assertEquals(15.0, Frete.calcular(10.0, 5.0));
    }
}
