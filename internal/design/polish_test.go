package design

import "testing"

func TestAuditPolish_Good(t *testing.T) {
	f := &Fundamentals{}
	content := `
		<div class="hero" style="background: linear-gradient(135deg, #1a365d, #2d3748);">
			<svg class="icon"><use href="#hero-icon"/></svg>
			<h1>Build Better</h1>
			<div class="card" style="box-shadow: 0 2px 4px rgba(0,0,0,0.1);">
				<img src="feature.jpg" class="wp-image-123" />
			</div>
			<div class="divider wave"></div>
			<a href="#" style="transition: all 0.3s ease;">Get Started</a>
		</div>
	`
	result := f.AuditPolish(content, "home")
	if !result.Pass {
		t.Errorf("Rich content should pass polish: issues=%v", result.Issues)
	}
}

func TestAuditPolish_Bare(t *testing.T) {
	f := &Fundamentals{}
	result := f.AuditPolish("<div><p>Hello world</p></div>", "home")
	if result.Pass {
		t.Error("Bare content should fail polish")
	}
	if len(result.Issues) == 0 {
		t.Error("Should have issues for bare content")
	}
}

func TestPolishPromptRules(t *testing.T) {
	f := &Fundamentals{}
	rules := f.PolishPromptRules()
	if rules == "" {
		t.Error("Polish prompt rules should not be empty")
	}
}

func TestValidateNavigation(t *testing.T) {
	f := &Fundamentals{}

	good := f.ValidateNavigation([]string{"Home", "Features", "Pricing", "About", "Contact"})
	if !good.Pass {
		t.Errorf("Good nav should pass: %v", good.Issues)
	}

	tooMany := f.ValidateNavigation([]string{"A", "B", "C", "D", "E", "F", "G", "H"})
	if tooMany.Pass {
		t.Error("8 nav items should fail (max 7)")
	}

	empty := f.ValidateNavigation([]string{})
	if empty.Pass {
		t.Error("Empty nav should fail")
	}
}

func TestValidateSitemap(t *testing.T) {
	f := &Fundamentals{}

	good := f.ValidateSitemap([]string{"Home", "About", "Features", "Pricing", "Contact"})
	if !good.Pass {
		t.Errorf("Good sitemap should pass: %v", good.Issues)
	}

	noHome := f.ValidateSitemap([]string{"Features", "Pricing"})
	if noHome.Pass {
		t.Error("Sitemap without Home should fail")
	}
}

func TestIAPromptRules(t *testing.T) {
	f := &Fundamentals{}
	rules := f.IAPromptRules()
	if rules == "" {
		t.Error("IA prompt rules should not be empty")
	}
}
