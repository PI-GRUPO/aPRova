import org.junit.Test;

import static org.junit.Assert.assertEquals;

public class AquecimentoTest {
    @Test
    public void soma() {
        assertEquals(3, Aquecimento.somar(1, 2));
    }
}
