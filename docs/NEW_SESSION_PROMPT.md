# Prompt for New Chat Session: LCC SDK Zero-Intrusion Refactoring

## 📋 Initial Prompt

```
请帮我完成 LCC SDK 的零侵入重构工作。

背景：
- lcc-demo-app 已完成零侵入设计的展示和文档（理想目标架构）
- lcc-sdk 仍然使用旧的侵入式 API，需要重构为真正的零侵入实现

任务文档位置：
/home/fila/jqdDev_2025/lcc-sdk/docs/REFACTORING_TASK_SDK_ZERO_INTRUSION.md

请先阅读这个任务文档，然后制定执行计划并开始实施。

关键要求：
1. SDK API 改为产品级（移除 featureID 参数）
2. 实现辅助函数注册系统
3. 保持向后兼容性
4. 所有代码注释用英文，与我交流用中文

工作目录：
/home/fila/jqdDev_2025/lcc-sdk

参考文档（Demo App）：
- /home/fila/jqdDev_2025/lcc-demo-app/docs/REFACTORING_COMPLETION_ZERO_INTRUSION.md
- /home/fila/jqdDev_2025/lcc-demo-app/docs/ZERO_INTRUSION_COMPARISON.md
```

---

## 📝 Context Summary

### What's Been Done (Demo App)
✅ **lcc-demo-app** - Completed refactoring to showcase zero-intrusion design:
- Moved limits from feature-level to product-level
- Updated all UI examples to show zero-intrusion approach
- Created comprehensive documentation
- Defined ideal YAML configuration format
- Demonstrated helper function system

**Key Files:**
- `/home/fila/jqdDev_2025/lcc-demo-app/docs/REFACTORING_COMPLETION_ZERO_INTRUSION.md` - Complete refactoring report
- `/home/fila/jqdDev_2025/lcc-demo-app/docs/ZERO_INTRUSION_COMPARISON.md` - Before/after comparison
- `/home/fila/jqdDev_2025/lcc-demo-app/internal/web/limits.go` - Updated limit examples with helper functions

### What Needs to Be Done (SDK)
⚠️ **lcc-sdk** - Core implementation work:

#### Current State (Bad)
```go
// Invasive API - requires featureID everywhere
client.Consume(featureID string, amount int, meta map[string]any)
client.CheckTPS(featureID string, currentTPS float64)
client.CheckCapacity(featureID string, currentUsed int)
client.AcquireSlot(featureID string, meta map[string]any)
```

#### Target State (Good)
```go
// Product-level API - no featureID needed
client.Consume(amount int) (bool, int, error)
client.CheckTPS() (bool, float64, error)
client.CheckCapacity(currentUsed int) (bool, int, error)
client.AcquireSlot() (ReleaseFunc, bool, error)

// Helper function system
client.RegisterHelpers(&HelperFunctions{
    QuotaConsumer:   myCustomAmountCalculator,
    TPSProvider:     myCustomTPSMeasure,
    CapacityCounter: myResourceCounter,
})
```

---

## 🎯 Task Breakdown

### Phase 1: API Refactoring (6 tasks)
1. Create `pkg/client/helpers.go` - Helper function types
2. Add helper registration to Client
3. Refactor `Consume()` API
4. Refactor `CheckTPS()` API  
5. Refactor `CheckCapacity()` API
6. Refactor `AcquireSlot()` API

### Phase 2: Internal TPS Tracking (1 task)
1. Create `pkg/client/tps_tracker.go` - Internal TPS measurement

### Phase 3: Code Generator (2 tasks)
1. Enhance YAML parser for product-level limits
2. Implement code generator with auto-injection

### Phase 4: Testing & Docs (3 tasks)
1. Unit tests
2. Integration tests
3. Documentation updates

---

## 📦 Key Files to Modify

### Existing Files
- `pkg/client/client.go` - Main refactoring (API changes)
- `pkg/config/types.go` - Add ProductLimits struct
- `pkg/codegen/generator.go` - Enhanced code generation

### New Files to Create
- `pkg/client/helpers.go` - Helper function system
- `pkg/client/tps_tracker.go` - Internal TPS tracking
- `docs/HELPER_FUNCTIONS_GUIDE.md` - User guide
- `docs/MIGRATION_GUIDE_V2.md` - Migration instructions
- `examples/zero-intrusion/` - Example project

---

## 🔍 Important Notes

### Design Principles
1. **Product-Level > Feature-Level**: All limits apply to entire product
2. **Zero-Intrusion**: Business logic has no license code
3. **Helper Functions**: Optional (Quota, TPS) vs Required (Capacity)
4. **Backward Compatibility**: Keep old APIs as `*Deprecated()`

### Helper Function Rules
- **QuotaConsumer**: Optional (default = 1 unit)
- **TPSProvider**: Optional (SDK auto-tracks)
- **CapacityCounter**: REQUIRED (SDK can't know your resources)
- **Concurrency**: No helper (SDK manages automatically)

### Code Standards
- All code comments in English
- All log messages in English
- Communication with user in Chinese
- Follow existing code style

---

## ✅ Success Criteria

When done, the SDK should:
- [ ] Have product-level API (no featureID parameters)
- [ ] Support helper function registration
- [ ] Maintain backward compatibility
- [ ] Pass all existing tests
- [ ] Have new tests for helper system
- [ ] Have updated documentation
- [ ] Include working examples

---

## 🚀 Getting Started

### Step 1: Understand Current State
```bash
cd /home/fila/jqdDev_2025/lcc-sdk
cat pkg/client/client.go | grep "func (c \*Client)"
```

### Step 2: Read Task Document
```bash
cat docs/REFACTORING_TASK_SDK_ZERO_INTRUSION.md
```

### Step 3: Review Demo App Design
```bash
cd /home/fila/jqdDev_2025/lcc-demo-app
cat docs/REFACTORING_COMPLETION_ZERO_INTRUSION.md
```

### Step 4: Start Implementation
Begin with Phase 1, Task 1.1 (Create helpers.go)

---

## 📚 Reference Links

### Documentation
- Task Spec: `/home/fila/jqdDev_2025/lcc-sdk/docs/REFACTORING_TASK_SDK_ZERO_INTRUSION.md`
- Demo Completion: `/home/fila/jqdDev_2025/lcc-demo-app/docs/REFACTORING_COMPLETION_ZERO_INTRUSION.md`
- Comparison Guide: `/home/fila/jqdDev_2025/lcc-demo-app/docs/ZERO_INTRUSION_COMPARISON.md`

### Code References
- Current Client API: `/home/fila/jqdDev_2025/lcc-sdk/pkg/client/client.go`
- Demo Limit Examples: `/home/fila/jqdDev_2025/lcc-demo-app/internal/web/limits.go`
- YAML Format: Demo app's `GetYAMLConfig()` in `products.go`

---

## 💡 Tips for AI Assistant

1. **Read Task Doc First**: Understand all phases before starting
2. **Create TODO List**: Use todo tools to track progress
3. **Incremental Changes**: Do one task at a time, test as you go
4. **Backward Compatibility**: Always keep old APIs with `Deprecated` suffix
5. **Test Everything**: Run tests after each major change
6. **Document Changes**: Update docs as you implement

---

## 🎓 Expected Workflow

1. AI reads this prompt and task document
2. AI creates a TODO list from Phase 1-4 tasks
3. AI starts with Phase 1, Task 1.1
4. AI implements each task, marks done, moves to next
5. AI runs tests after each phase
6. AI updates documentation
7. AI creates final summary report

---

**Good luck! The zero-intrusion future awaits! 🚀**
