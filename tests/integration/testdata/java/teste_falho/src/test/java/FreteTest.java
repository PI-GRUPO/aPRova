import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.assertEquals;

class FreteTest {
    @Test
    void calculaFreteCorretamente() {
        assertEquals(15.0, Frete.calcular(10.0, 5.0));
    }

    @Test
    void calculaFreteGratis() {
        assertEquals(12.0, Frete.calcular(10.0, 0.0));
    }
}
