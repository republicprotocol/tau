package algebra

// The FieldElement interface defines the required methods for an object to be
// the element of a field. Convenience subtraction and division methods can be
// defined using the respective addidtive and multiplicative inverses.
type FieldElement interface {
	Add(FieldElement, FieldElement)
	Neg(FieldElement)
	Mul(FieldElement, FieldElement)
	MulInv(FieldElement)
}
