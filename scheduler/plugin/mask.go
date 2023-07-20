package wasm

import (
	"errors"
	"io"

	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type interfaces uint

const (
	iEnqueueExtensions interfaces = 1 << iota
	iPreFilterExtensions
	iPreFilterPlugin
	iFilterPlugin
	iPostFilterPlugin
	iPreScorePlugin
	iScoreExtensions
	iScorePlugin
	iReservePlugin
	iPermitPlugin
	iPreBindPlugin
	iBindPlugin
	iPostBindPlugin
)

// maskInterfaces ensures the caller can do type checking to detect what the
// plugin supports.
//
// It isn't feasible to do fine-grained checks for all interfaces, as there are
// 13. This would be 2^13 or 8192 types of interfaces. Instead, this generates
// a type for each main plugin, e.g. scorePlugin.
//
// Unless something changes, this means you cannot declare a plugin that only
// does a "pre" stage. e.g. a framework.PreScorePlugin that isn't also a
// framework.ScorePlugin. Only exceptions are documented below:
//
//   - framework.PreFilterPlugin is always implemented, because this is used to
//     reset cycle state.
func maskInterfaces(plugin *wasmPlugin) (framework.Plugin, error) {
	// First, mask all interfaces that are coupled together
	i := plugin.guestInterfaces & ^(iEnqueueExtensions |
		iPreFilterExtensions |
		iPreFilterPlugin |
		iPostFilterPlugin |
		iPreScorePlugin |
		iScoreExtensions |
		iPreBindPlugin |
		iPostBindPlugin)

	switch i {
	case iFilterPlugin:
		type filter interface {
			basePlugin
			filterPlugin
		}
		return struct{ filter }{plugin}, nil
	case iFilterPlugin | iScorePlugin:
		type filterScore interface {
			basePlugin
			filterPlugin
			scorePlugin
		}
		return struct{ filterScore }{plugin}, nil
	case iFilterPlugin | iScorePlugin | iBindPlugin:
		type filterScorePreBind interface {
			basePlugin
			filterPlugin
			scorePlugin
			bindPlugin
		}
		return struct{ filterScorePreBind }{plugin}, nil
	case iReservePlugin:
		type filterReserve interface {
			basePlugin
			reservePlugin
		}
		return struct{ filterReserve }{plugin}, nil
	case iPermitPlugin:
		type filterPermit interface {
			basePlugin
			permitPlugin
		}
		return struct{ filterPermit }{plugin}, nil
	case iBindPlugin:
		type filterBind interface {
			basePlugin
			bindPlugin
		}
		return struct{ filterBind }{plugin}, nil
	case iFilterPlugin | iReservePlugin:
		type filterReserve interface {
			basePlugin
			filterPlugin
			reservePlugin
		}
		return struct{ filterReserve }{plugin}, nil
	case iFilterPlugin | iPermitPlugin:
		type filterPermit interface {
			basePlugin
			filterPlugin
			permitPlugin
		}
		return struct{ filterPermit }{plugin}, nil
	case iFilterPlugin | iBindPlugin:
		type filterBind interface {
			basePlugin
			filterPlugin
			bindPlugin
		}
		return struct{ filterBind }{plugin}, nil
	case iScorePlugin:
		type score interface {
			basePlugin
			scorePlugin
		}
		return struct{ score }{plugin}, nil
	case iScorePlugin | iReservePlugin:
		type scoreReserve interface {
			basePlugin
			scorePlugin
			reservePlugin
		}
		return struct{ scoreReserve }{plugin}, nil
	case iScorePlugin | iPermitPlugin:
		type scorePermit interface {
			basePlugin
			scorePlugin
			permitPlugin
		}
		return struct{ scorePermit }{plugin}, nil
	case iScorePlugin | iBindPlugin:
		type scoreBind interface {
			basePlugin
			scorePlugin
			bindPlugin
		}
		return struct{ scoreBind }{plugin}, nil
	case iReservePlugin | iBindPlugin:
		type reserveBind interface {
			basePlugin
			reservePlugin
			bindPlugin
		}
		return struct{ reserveBind }{plugin}, nil
	case iPermitPlugin | iBindPlugin:
		type permitBind interface {
			basePlugin
			permitPlugin
			bindPlugin
		}
		return struct{ permitBind }{plugin}, nil
	case iFilterPlugin | iScorePlugin | iReservePlugin:
		type filterScoreReserve interface {
			basePlugin
			filterPlugin
			scorePlugin
			reservePlugin
		}
		return struct{ filterScoreReserve }{plugin}, nil
	case iFilterPlugin | iScorePlugin | iPermitPlugin:
		type filterScorePermit interface {
			basePlugin
			filterPlugin
			scorePlugin
			permitPlugin
		}
		return struct{ filterScorePermit }{plugin}, nil
	case iFilterPlugin | iScorePlugin | iBindPlugin | iReservePlugin:
		type filterScoreBindReserve interface {
			basePlugin
			filterPlugin
			scorePlugin
			bindPlugin
			reservePlugin
		}
		return struct{ filterScoreBindReserve }{plugin}, nil
	case iFilterPlugin | iScorePlugin | iBindPlugin | iPermitPlugin:
		type filterScoreBindPermit interface {
			basePlugin
			filterPlugin
			scorePlugin
			bindPlugin
			permitPlugin
		}
		return struct{ filterScoreBindPermit }{plugin}, nil
	case iScorePlugin | iReservePlugin | iBindPlugin:
		type scoreReserveBind interface {
			basePlugin
			scorePlugin
			reservePlugin
			bindPlugin
		}
		return struct{ scoreReserveBind }{plugin}, nil
	case iScorePlugin | iPermitPlugin | iBindPlugin:
		type scorePermitBind interface {
			basePlugin
			scorePlugin
			permitPlugin
			bindPlugin
		}
		return struct{ scorePermitBind }{plugin}, nil
	case iReservePlugin | iPermitPlugin | iBindPlugin:
		type reservePermitBind interface {
			basePlugin
			reservePlugin
			permitPlugin
			bindPlugin
		}
		return struct{ reservePermitBind }{plugin}, nil
	case iFilterPlugin | iScorePlugin | iReservePlugin | iBindPlugin | iPermitPlugin:
		type filterScoreReservePermitBind interface {
			basePlugin
			filterPlugin
			scorePlugin
			reservePlugin
			permitPlugin
			bindPlugin
		}
		return struct{ filterScoreReservePermitBind }{plugin}, nil
	}

	// Handle special cases
	switch plugin.guestInterfaces {
	case iPreFilterPlugin: // Special-cased form of filter.
		return struct{ basePlugin }{plugin}, nil
	default:
		return nil, errors.New("filter, score, reserve, permit or bind must be exported")
	}
}

type basePlugin interface {
	framework.EnqueueExtensions
	framework.PreFilterPlugin // to implement cycle state reset
	io.Closer
	ProfilerSupport
}

type filterPlugin interface {
	framework.PreFilterExtensions
	framework.PreFilterPlugin
	framework.FilterPlugin
	framework.PostFilterPlugin
}

type scorePlugin interface {
	framework.PreScorePlugin
	framework.ScoreExtensions
	framework.ScorePlugin
}

type reservePlugin interface {
	framework.ReservePlugin
}

type permitPlugin interface {
	framework.PermitPlugin
}

type bindPlugin interface {
	framework.PreBindPlugin
	framework.BindPlugin
	framework.PostBindPlugin
}
