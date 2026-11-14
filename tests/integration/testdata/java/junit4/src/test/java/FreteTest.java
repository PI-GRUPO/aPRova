import org.junit.Test;

import static org.junit.Assert.assertEquals;

public class FreteTest {
    @Test
    public void calculaFreteComTaxa() {
        assertEquals(15.0, Frete.calcular(10.0, 5.0), 0.0);
    }

    @Test
    public void calculaFreteGratis() {
        assertEquals(10.0, Frete.calcular(10.0, 0.0), 0.0);
    }
}
