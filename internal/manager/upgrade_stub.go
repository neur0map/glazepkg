package manager

func (a *AUR) UpgradePackage(string) error                     { return ErrUpgradeNotSupported }
func (p *Pipx) UpgradePackage(string) error                    { return ErrUpgradeNotSupported }
func (g *Go) UpgradePackage(string) error                      { return ErrUpgradeNotSupported }
func (p *Pnpm) UpgradePackage(string) error                    { return ErrUpgradeNotSupported }
func (b *Bun) UpgradePackage(string) error                     { return ErrUpgradeNotSupported }
func (f *Flatpak) UpgradePackage(string) error                 { return ErrUpgradeNotSupported }
func (m *MacPorts) UpgradePackage(string) error                { return ErrUpgradeNotSupported }
func (p *Pkgsrc) UpgradePackage(string) error                  { return ErrUpgradeNotSupported }
func (o *Opam) UpgradePackage(string) error                    { return ErrUpgradeNotSupported }
func (g *Gem) UpgradePackage(string) error                     { return ErrUpgradeNotSupported }
func (f *FreeBSDPkg) UpgradePackage(string) error              { return ErrUpgradeNotSupported }
func (c *Composer) UpgradePackage(string) error                { return ErrUpgradeNotSupported }
func (m *Mas) UpgradePackage(string) error                     { return ErrUpgradeNotSupported }
func (a *Apk) UpgradePackage(string) error                     { return ErrUpgradeNotSupported }
func (n *Nix) UpgradePackage(string) error                     { return ErrUpgradeNotSupported }
func (c *Conda) UpgradePackage(string) error                   { return ErrUpgradeNotSupported }
func (l *Luarocks) UpgradePackage(string) error                { return ErrUpgradeNotSupported }
func (x *Xbps) UpgradePackage(string) error                    { return ErrUpgradeNotSupported }
func (p *Portage) UpgradePackage(string) error                 { return ErrUpgradeNotSupported }
func (g *Guix) UpgradePackage(string) error                    { return ErrUpgradeNotSupported }
func (n *Nuget) UpgradePackage(string) error                   { return ErrUpgradeNotSupported }
func (p *PowerShell) UpgradePackage(string) error              { return ErrUpgradeNotSupported }
func (w *WindowsUpdates) UpgradePackage(string) error          { return ErrUpgradeNotSupported }
