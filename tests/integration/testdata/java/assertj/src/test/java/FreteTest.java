import org.junit.jupiter.api.Test;

import static org.assertj.core.api.Assertions.assertThat;

class FreteTest {
    @Test
    void calculaFreteComTaxa() {
        assertThat(Frete.calcular(10.0, 5.0)).isEqualTo(15.0);
    }

    @Test
    void calculaFreteGratis() {
        assertThat(Frete.calcular(10.0, 0.0)).isEqualTo(10.0);
    }
}
