package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}
func modPow(base, exp, mod *big.Int) *big.Int {
	return new(big.Int).Exp(base, exp, mod)
}

func extGCD(a, b *big.Int) (gcd, x, y *big.Int) {
	if b.Sign() == 0 {
		return new(big.Int).Set(a), big.NewInt(1), big.NewInt(0)
	}
	g, x1, y1 := extGCD(b, new(big.Int).Mod(a, b))
	q := new(big.Int).Div(a, b)
	x = new(big.Int).Set(y1)
	y = new(big.Int).Sub(x1, new(big.Int).Mul(q, y1))
	return g, x, y
}

func isPrime(n int64) bool {
	if n < 2 {
		return false
	}
	return big.NewInt(n).ProbablyPrime(20)
}

func rabinEncrypt(m, b, n *big.Int) *big.Int {
	mb := new(big.Int).Add(m, b)
	c := new(big.Int).Mul(m, mb)
	return c.Mod(c, n)
}

func sqrtModPrime(D, p *big.Int) *big.Int {
	exp := new(big.Int).Add(p, big.NewInt(1))
	exp.Div(exp, big.NewInt(4))
	return modPow(D, exp, p)
}

func rabinCandidates(c, b, p, q, n *big.Int) []*big.Int {
	D := new(big.Int).Mul(b, b)
	D.Add(D, new(big.Int).Mul(big.NewInt(4), c))
	D.Mod(D, n)

	Dp := new(big.Int).Mod(D, p)
	Dq := new(big.Int).Mod(D, q)

	mp := sqrtModPrime(Dp, p)
	mq := sqrtModPrime(Dq, q)

	_, yp, yq := extGCD(p, q)

	term1 := new(big.Int).Mul(yp, new(big.Int).Mul(p, mq))
	term2 := new(big.Int).Mul(yq, new(big.Int).Mul(q, mp))

	modN := func(x *big.Int) *big.Int {
		r := new(big.Int).Mod(x, n)
		if r.Sign() < 0 {
			r.Add(r, n)
		}
		return r
	}
	r1 := modN(new(big.Int).Add(term1, term2))
	r2 := modN(new(big.Int).Sub(n, r1))
	r3 := modN(new(big.Int).Sub(term1, term2))
	r4 := modN(new(big.Int).Sub(n, r3))

	twoInv := new(big.Int).ModInverse(big.NewInt(2), n)

	seen := map[string]bool{}
	var candidates []*big.Int

	for _, d := range []*big.Int{r1, r2, r3, r4} {
		m := new(big.Int).Sub(d, b)
		m.Mod(m, n)
		if m.Sign() < 0 {
			m.Add(m, n)
		}
		m.Mul(m, twoInv)
		m.Mod(m, n)

		key := m.String()
		if !seen[key] {
			seen[key] = true
			candidates = append(candidates, m)
		}
	}

	return candidates
}

func selectRoot(c, b, n *big.Int, candidates []*big.Int) (byte, bool) {
	limit := big.NewInt(256)

	for _, m := range candidates {
		if m.Sign() < 0 || m.Cmp(limit) >= 0 {
			continue
		}
		if rabinEncrypt(m, b, n).Cmp(c) == 0 {
			return byte(m.Int64()), true
		}
	}
	return 0, false
}

func validateParams(pVal, qVal, bVal int64) error {
	if pVal < 3 {
		return fmt.Errorf("p должно быть ≥ 3")
	}
	if qVal < 3 {
		return fmt.Errorf("q должно быть ≥ 3")
	}
	if pVal == qVal {
		return fmt.Errorf("p и q должны быть различными")
	}
	if !isPrime(pVal) {
		return fmt.Errorf("p = %d не является простым числом", pVal)
	}
	if !isPrime(qVal) {
		return fmt.Errorf("q = %d не является простым числом", qVal)
	}
	if pVal%4 != 3 {
		return fmt.Errorf("p = %d: нужно p ≡ 3 mod 4 (остаток %d)", pVal, pVal%4)
	}
	if qVal%4 != 3 {
		return fmt.Errorf("q = %d: нужно q ≡ 3 mod 4 (остаток %d)", qVal, qVal%4)
	}
	n := pVal * qVal
	if n <= 256 {
		return fmt.Errorf("n = p×q = %d должно быть > 256", n)
	}
	if bVal <= 0 {
		return fmt.Errorf("b должно быть > 0")
	}
	if bVal >= n {
		return fmt.Errorf("b = %d должно быть < n = %d", bVal, n)
	}
	return nil
}

func formatInts(nums []int64, maxShow int) string {
	if len(nums) == 0 {
		return "(пусто)"
	}
	total := len(nums)
	end := total
	if end > maxShow {
		end = maxShow
	}
	parts := make([]string, end)
	for i := 0; i < end; i++ {
		parts[i] = strconv.FormatInt(nums[i], 10)
	}
	s := strings.Join(parts, " ")
	if total > maxShow {
		s += fmt.Sprintf(" ...(%d чисел всего)", total)
	}
	return s
}

func (a *App) EncryptFile(
	pVal, qVal, bVal int,
	fileData []int,
	filename string,
) (map[string]interface{}, error) {

	if err := validateParams(int64(pVal), int64(qVal), int64(bVal)); err != nil {
		return nil, err
	}

	p := big.NewInt(int64(pVal))
	q := big.NewInt(int64(qVal))
	b := big.NewInt(int64(bVal))
	n := new(big.Int).Mul(p, q)

	data := make([]byte, len(fileData))
	for i, v := range fileData {
		data[i] = byte(v & 0xFF)
	}
	encrypted := make([]uint64, len(data))
	for i, byt := range data {
		m := big.NewInt(int64(byt))
		c := rabinEncrypt(m, b, n)
		encrypted[i] = c.Uint64()
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("ошибка директории: %v", err)
	}
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filepath.Base(filename), ext)
	base = strings.TrimSuffix(base, "_decrypted")
	base = strings.TrimSuffix(base, "_encrypted")
	outName := base + "_encrypted" + ext
	outPath := filepath.Join(currentDir, outName)

	f, err := os.Create(outPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания файла: %v", err)
	}
	defer f.Close()

	buf8 := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf8, uint64(len(encrypted)))
	if _, err := f.Write(buf8); err != nil {
		return nil, fmt.Errorf("ошибка записи count: %v", err)
	}

	for _, v := range encrypted {
		binary.LittleEndian.PutUint64(buf8, v)
		if _, err := f.Write(buf8); err != nil {
			return nil, fmt.Errorf("ошибка записи данных: %v", err)
		}
	}

	origNums := make([]int64, len(data))
	for i, v := range data {
		origNums[i] = int64(v)
	}
	encNums := make([]int64, len(encrypted))
	for i, v := range encrypted {
		encNums[i] = int64(v)
	}

	return map[string]interface{}{
		"original_bytes":  formatInts(origNums, 60),
		"encrypted_bytes": formatInts(encNums, 60),
		"saved_as":        outName,
		"n_value":         n.String(),
	}, nil
}

func (a *App) DecryptFile(
	pVal, qVal, bVal int,
	fileData []int,
	filename string,
) (map[string]interface{}, error) {

	if err := validateParams(int64(pVal), int64(qVal), int64(bVal)); err != nil {
		return nil, err
	}

	p := big.NewInt(int64(pVal))
	q := big.NewInt(int64(qVal))
	b := big.NewInt(int64(bVal))
	n := new(big.Int).Mul(p, q)

	raw := make([]byte, len(fileData))
	for i, v := range fileData {
		raw[i] = byte(v & 0xFF)
	}

	if len(raw) < 8 {
		return nil, fmt.Errorf(
			"файл слишком мал (%d байт). Выберите файл _encrypted", len(raw),
		)
	}

	count := int(binary.LittleEndian.Uint64(raw[0:8]))

	expected := 8 + count*8
	if len(raw) != expected {
		return nil, fmt.Errorf(
			"неверный размер файла: ожидалось %d байт (count=%d), получено %d.\n"+
				"Выберите файл _encrypted, созданный этой программой.",
			expected, count, len(raw),
		)
	}
	encryptedVals := make([]*big.Int, count)
	for i := 0; i < count; i++ {
		offset := 8 + i*8
		v := binary.LittleEndian.Uint64(raw[offset : offset+8])
		encryptedVals[i] = new(big.Int).SetUint64(v)
	}

	decrypted := make([]byte, count)
	failCount := 0
	failExamples := []string{}

	for i, c := range encryptedVals {
		candidates := rabinCandidates(c, b, p, q, n)
		byt, ok := selectRoot(c, b, n, candidates)
		if ok {
			decrypted[i] = byt
		} else {
			failCount++
			if len(failExamples) < 5 {
				cs := make([]string, len(candidates))
				for j, cand := range candidates {
					cs[j] = cand.String()
				}
				failExamples = append(failExamples,
					fmt.Sprintf("байт#%d c=%s кандидаты=[%s]",
						i, c.String(), strings.Join(cs, ",")))
			}
		}
	}

	if failCount > 0 {
		return nil, fmt.Errorf(
			"не удалось расшифровать %d байт из %d.\n%s\n"+
				"Проверьте параметры p=%d q=%d b=%d",
			failCount, count,
			strings.Join(failExamples, "\n"),
			pVal, qVal, bVal,
		)
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("ошибка директории: %v", err)
	}
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filepath.Base(filename), ext)
	base = strings.TrimSuffix(base, "_encrypted")
	base = strings.TrimSuffix(base, "_decrypted")
	outName := base + "_decrypted" + ext
	outPath := filepath.Join(currentDir, outName)

	if err := os.WriteFile(outPath, decrypted, 0644); err != nil {
		return nil, fmt.Errorf("ошибка сохранения: %v", err)
	}

	encNums := make([]int64, len(encryptedVals))
	for i, v := range encryptedVals {
		encNums[i] = v.Int64()
	}
	decNums := make([]int64, len(decrypted))
	for i, v := range decrypted {
		decNums[i] = int64(v)
	}

	return map[string]interface{}{
		"original_bytes":  formatInts(encNums, 60),
		"encrypted_bytes": formatInts(decNums, 60),
		"saved_as":        outName,
	}, nil
}
