#
# Copyright (C) 2024 OpenWrt.org
#
# This is free software, licensed under the GNU General Public License v2.
#

include $(TOPDIR)/rules.mk

LUCI_TITLE:=LuCI support for PanSou
LUCI_DEPENDS:=+pansou-openwrt +luci-base
LUCI_PKGARCH:=all

PKG_NAME:=luci-app-pansou
PKG_VERSION:=1.0.0
PKG_RELEASE:=1

PKG_LICENSE:=GPL-2.0
PKG_MAINTAINER:=YourName <your@email.com>

include $(TOPDIR)/feeds/luci/luci.mk

# call BuildPackage - OpenWrt buildroot signature
