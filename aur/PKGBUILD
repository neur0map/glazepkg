# Maintainer: neur0map <90535780+neur0map@users.noreply.github.com>
pkgname=gpk-bin
pkgver=0.2.5
pkgrel=1
pkgdesc="TUI dashboard that unifies 34 package managers into one searchable view"
arch=('x86_64' 'aarch64')
url="https://github.com/neur0map/glazepkg"
license=('GPL-3.0-or-later')
depends=()
provides=('gpk')
conflicts=('gpk')
source_x86_64=("${url}/releases/download/v${pkgver}/gpk-linux-amd64")
source_aarch64=("${url}/releases/download/v${pkgver}/gpk-linux-arm64")
sha256sums_x86_64=('b642280351610dda2c9470620631756fa28d477bbf7051292fc79815a3ee465f')
sha256sums_aarch64=('71ac2fd81b7e2d3811f49a7503b1bf73383bf7fa696b3daf91bf8a6099d8811b')

package() {
    if [[ "$CARCH" == "x86_64" ]]; then
        install -Dm755 "gpk-linux-amd64" "${pkgdir}/usr/bin/gpk"
    elif [[ "$CARCH" == "aarch64" ]]; then
        install -Dm755 "gpk-linux-arm64" "${pkgdir}/usr/bin/gpk"
    fi
}
