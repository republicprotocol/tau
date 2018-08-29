package algebra_test

import (
	"math/big"
	"math/rand"

	. "github.com/onsi/ginkgo/extensions/table"

	. "github.com/republicprotocol/smpc-go/core/vss/algebra"
)

// RandomBool returns a random boolean with equal probability.
func RandomBool() bool {
	return rand.Float32() < 0.5
}

// RandomNotInField will create a random integer that is not in the given
// field. It will, with equal probability, pick an integer either too large
// (between prime and 2*prime) or too small (a negative integer in the range
// -prime to -1).
func RandomNotInField(field *Fp) (r *big.Int) {
	r = field.Random()

	if RandomBool() {
		// Make r too small
		r.Neg(r)

		// Subtract 1 in case r was 0
		r.Sub(r, big.NewInt(1))
	} else {
		// Make r too big
		if r.Sign() == 0 {
			// Ensure that r is not 0
			r.Add(r, big.NewInt(1))
		}
		addinv := big.NewInt(0).Set(r)
		field.Neg(addinv, addinv)
		r.Add(r, big.NewInt(0).Add(r, addinv))
	}

	return
}

// PrimeEntries is a list of table entries of random prime numbers less than
// 2^64
var PrimeEntries = []TableEntry{
	Entry("for the given prime", big.NewInt(2)),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(11415648579556416673))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(10891814531730287201))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(2173186581265841101))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(8037833094411151351))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(160889637713534993))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(2598439422723623851))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(15063151627087255057))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(5652006400289677651))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(1075037556033023437))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(4383237663223642961))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(11491288605849083743))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(18060401258323832179))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(2460931945023125813))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(14720243597953921717))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(11460698326622148979))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(7289555056001917459))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(10819520547428938847))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(17087033667620041241))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(11897098582950941981))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(14162389779744880153))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(3341353876108302833))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(2421057993123425237))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(6099033893113295747))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(9119102700930783271))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(11701444041617194927))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(6492121780466656261))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(1719187971393348791))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(7128898183300867241))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(10448609340017805841))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(5250106197074512951))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(12523635873138238501))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(6179856695580003673))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(14312226640074246223))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(2656168198924335947))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(15282215154228341597))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(5862491744359797091))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(10930389297127849337))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(15453819937382700221))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(8587765603082695229))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(6499635665205708017))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(9522904300687004989))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(6754377453775717483))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(10278941889065878913))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(4119057578904911521))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(2695278052346845627))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(2898709949625550547))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(14640846616444411459))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(8775965213363272289))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(7695258118026415753))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(9112974089849462297))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(14662204281882267989))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(4999606432544782237))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(8961999239135894533))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(14602672531347032081))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(14606570603637462067))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(3662715635181767911))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(15528677330235002987))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(17549052314223638287))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(14793342612719440001))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(1110258828067568087))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(8321432222762641111))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(2099085051126463573))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(17684615516776485691))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(5581192723150425841))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(12295043986397223823))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(4590971551517707183))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(6667954438606055873))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(11257624651846941287))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(11269427064747885857))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(10832662390615802801))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(1149178208693899297))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(7776311754824701427))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(12138619704493513207))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(11715817198039041233))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(8776823877387205793))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(900483851285056369))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(10565010275733687859))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(3598475899888315571))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(609292139725849487))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(2512663778109890407))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(5356705606915059847))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(4926920292130371833))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(15588936261527250763))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(17674364459850493807))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(15010913622986786653))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(17165846626530660623))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(13953789782321853637))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(9875187539480118827))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(9411830831698978339))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(2181702112484780533))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(15314636212339236139))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(511205612465019343))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(8113765242226142771))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(8891182210143952699))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(6315655006279877437))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(8364339317215443659))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(1207853845318533811))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(11869971765257449303))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(17490095259054169019))),
	Entry("for the given prime", big.NewInt(0).SetUint64(uint64(7590272435001495331))),
}

// CompositeEntries is a list of table entries of random composite numbers less
// than 2^64
var CompositeEntries = []TableEntry{
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(2128090164445538166))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(17364939545239290576))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(1391821019477845399))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(16344437384279108147))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(2706066384079165076))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(263258624498915050))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(14818061775102548121))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(1952946230500555180))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(1533376888302800451))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(17809671752350070514))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(10364531498445533344))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(14273206633946995539))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(12643952213924983463))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(2146366126026109200))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(17809296478810548798))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(11905138142927281665))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(12035297787850296595))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(15772059672965580703))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(12232115969293837225))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(7537506351955809400))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(3425696715341053332))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(9709238070217535437))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(1935494489933823319))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(12612268782559485113))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(18159541596081065346))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(10759464702836858751))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(13728453529377421007))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(69418916488692231))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(456948175610779306))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(17950920828782482074))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(9901170790800645069))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(12641866484572220365))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(7518617566440586766))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(10785938751583250077))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(14887799717827156617))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(8476817532120081616))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(8213332099789609135))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(17478036555002556292))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(16452353133078716214))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(11229534316970022284))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(2249246575181508387))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(16701353593969359798))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(2268846483146368570))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(15216106240138036671))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(3629274280245081699))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(8838547407473940700))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(14607019161453060166))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(10933892343759177876))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(17533994693110343643))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(17527878693134563808))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(12031659812875835128))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(13171104285895938330))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(13518243952715655412))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(14088799075017693502))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(5915590918833620772))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(8534771589081599521))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(11740659755464401986))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(9125744824015575765))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(10640386713011311188))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(4192918514089713520))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(13083732921183804232))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(142355766992216147))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(964162564491293272))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(10862457803932743101))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(9188526282721813619))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(3605114845807365787))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(12407297878231536830))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(16255018895109265877))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(7799995483122850831))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(4024049630673895166))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(17987114619304905091))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(3335865492762087304))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(16052392637630112596))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(15948857855315263255))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(15100230438765809012))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(16132807435522779839))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(13192973676941210129))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(1762389507026922365))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(6850486779606755972))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(11345745673597234178))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(4518955311280269129))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(11047962582926896977))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(14761002308622279574))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(791035351342998838))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(3003445626881514592))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(18204306655822160961))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(8035970954127497034))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(17299873097928164257))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(6432559618489345267))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(9789367420576356493))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(15533531660777583294))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(17224122940233208984))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(14420099037837298808))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(2419601594567570313))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(16210241368823343374))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(16601785937907502254))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(6134613613158864962))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(11425169933133155858))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(18032051221828905166))),
	Entry("for given composite", big.NewInt(0).SetUint64(uint64(4794593749443992175))),
}
