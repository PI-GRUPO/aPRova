import org.junit.jupiter.api.Test;

import static org.assertj.core.api.Assertions.assertThat;
import static org.junit.jupiter.api.Assertions.assertEquals;

class AquecimentoTest {
    @Test
    void soma() {
        assertEquals(3, Aquecimento.somar(1, 2));
        assertThat(Aquecimento.somar(2, 2)).isEqualTo(4);
    }
}
