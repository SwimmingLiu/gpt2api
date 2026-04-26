<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import * as poolApi from '@/api/accountPools'
import * as accountApi from '@/api/accounts'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()
const canWrite = computed(() => userStore.hasPerm('account:write'))

const loading = ref(false)
const rows = ref<poolApi.AccountPool[]>([])
const members = ref<poolApi.AccountPoolMember[]>([])
const accounts = ref<accountApi.Account[]>([])
const currentPoolID = ref<number>(0)

const poolDlg = ref(false)
const editingPoolID = ref(0)
const poolForm = reactive({
  code: '',
  name: '',
  pool_type: 'mixed',
  description: '',
  enabled: true,
  dispatch_strategy: 'least_recently_used',
  sticky_ttl_sec: 0,
})

const memberDlg = ref(false)
const memberForm = reactive({
  account_id: 0,
  enabled: true,
  weight: 100,
  priority: 100,
  max_parallel: 1,
  note: '',
})

async function loadPools() {
  loading.value = true
  try {
    const d = await poolApi.listAccountPools()
    rows.value = d.items || []
    if (!currentPoolID.value && rows.value.length > 0) {
      currentPoolID.value = rows.value[0].id
    }
  } finally {
    loading.value = false
  }
  await loadMembers()
}

async function loadMembers() {
  if (!currentPoolID.value) {
    members.value = []
    return
  }
  const d = await poolApi.listPoolMembers(currentPoolID.value)
  members.value = d.items || []
}

async function loadAccounts() {
  const d = await accountApi.listAccounts({ page: 1, page_size: 1000 })
  accounts.value = d.list || []
}

function openCreatePool() {
  editingPoolID.value = 0
  Object.assign(poolForm, {
    code: '',
    name: '',
    pool_type: 'mixed',
    description: '',
    enabled: true,
    dispatch_strategy: 'least_recently_used',
    sticky_ttl_sec: 0,
  })
  poolDlg.value = true
}

function openEditPool(row: poolApi.AccountPool) {
  editingPoolID.value = row.id
  Object.assign(poolForm, {
    code: row.code,
    name: row.name,
    pool_type: row.pool_type,
    description: row.description,
    enabled: row.enabled,
    dispatch_strategy: row.dispatch_strategy,
    sticky_ttl_sec: row.sticky_ttl_sec,
  })
  poolDlg.value = true
}

async function submitPool() {
  try {
    if (editingPoolID.value) {
      await poolApi.updateAccountPool(editingPoolID.value, poolForm)
      ElMessage.success('账号池已更新')
    } else {
      await poolApi.createAccountPool(poolForm)
      ElMessage.success('账号池已创建')
    }
    poolDlg.value = false
    await loadPools()
  } catch (e: any) {
    ElMessage.error(e?.message || '保存失败')
  }
}

async function onDeletePool(row: poolApi.AccountPool) {
  try {
    await ElMessageBox.confirm(`确定删除账号池「${row.name}」?`, '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  await poolApi.deleteAccountPool(row.id)
  ElMessage.success('账号池已删除')
  if (currentPoolID.value === row.id) currentPoolID.value = 0
  await loadPools()
}

function openAddMember() {
  Object.assign(memberForm, {
    account_id: 0,
    enabled: true,
    weight: 100,
    priority: 100,
    max_parallel: 1,
    note: '',
  })
  memberDlg.value = true
}

async function submitMember() {
  if (!currentPoolID.value || !memberForm.account_id) {
    ElMessage.warning('请选择账号')
    return
  }
  await poolApi.createPoolMember(currentPoolID.value, memberForm)
  ElMessage.success('成员已加入账号池')
  memberDlg.value = false
  await loadMembers()
}

async function onDeleteMember(row: poolApi.AccountPoolMember) {
  if (!currentPoolID.value) return
  await poolApi.deletePoolMember(currentPoolID.value, row.id)
  ElMessage.success('成员已移出账号池')
  await loadMembers()
}

const currentPool = computed(() => rows.value.find((x) => x.id === currentPoolID.value))
const accountMap = computed(() => {
  const map = new Map<number, accountApi.Account>()
  for (const a of accounts.value) map.set(a.id, a)
  return map
})

onMounted(async () => {
  await Promise.all([loadAccounts(), loadPools()])
})
</script>

<template>
  <div class="page-container">
    <div class="card-block">
      <div class="flex-between">
        <div>
          <h2 class="page-title">账号池管理</h2>
          <div class="page-sub">管理账号池、池类型和池成员。Phase 1 先提供基础 CRUD 和成员绑定。</div>
        </div>
        <el-button v-if="canWrite" type="primary" @click="openCreatePool">新建账号池</el-button>
      </div>
    </div>

    <div class="split-grid">
      <div class="card-block">
        <div class="section-title">账号池列表</div>
        <el-table v-loading="loading" :data="rows" row-key="id" highlight-current-row @current-change="(row:any) => { currentPoolID = row?.id || 0; loadMembers() }">
          <el-table-column prop="code" label="池编码" min-width="160" />
          <el-table-column prop="name" label="名称" min-width="140" />
          <el-table-column prop="pool_type" label="类型" width="100" />
          <el-table-column label="启用" width="80">
            <template #default="{ row }">
              <el-tag :type="row.enabled ? 'success' : 'info'" size="small">{{ row.enabled ? '启用' : '停用' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column v-if="canWrite" label="操作" width="120">
            <template #default="{ row }">
              <el-button link size="small" type="primary" @click.stop="openEditPool(row)">编辑</el-button>
              <el-button link size="small" type="danger" @click.stop="onDeletePool(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div class="card-block">
        <div class="flex-between">
          <div class="section-title">池成员{{ currentPool ? ` · ${currentPool.name}` : '' }}</div>
          <el-button v-if="canWrite" :disabled="!currentPoolID" @click="openAddMember">加入账号</el-button>
        </div>
        <el-table :data="members" row-key="id">
          <el-table-column label="账号" min-width="180">
            <template #default="{ row }">
              {{ accountMap.get(row.account_id)?.email || `#${row.account_id}` }}
            </template>
          </el-table-column>
          <el-table-column prop="weight" label="权重" width="80" />
          <el-table-column prop="priority" label="优先级" width="90" />
          <el-table-column prop="max_parallel" label="并发" width="80" />
          <el-table-column label="启用" width="80">
            <template #default="{ row }">
              <el-tag :type="row.enabled ? 'success' : 'info'" size="small">{{ row.enabled ? '启用' : '停用' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="note" label="备注" min-width="120" />
          <el-table-column v-if="canWrite" label="操作" width="80">
            <template #default="{ row }">
              <el-button link size="small" type="danger" @click="onDeleteMember(row)">移除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <el-dialog v-model="poolDlg" :title="editingPoolID ? '编辑账号池' : '新建账号池'" width="640px">
      <el-form label-width="120px">
        <el-form-item label="池编码"><el-input v-model="poolForm.code" :disabled="!!editingPoolID" /></el-form-item>
        <el-form-item label="名称"><el-input v-model="poolForm.name" /></el-form-item>
        <el-form-item label="类型">
          <el-select v-model="poolForm.pool_type" style="width: 180px">
            <el-option label="mixed" value="mixed" />
            <el-option label="chat" value="chat" />
            <el-option label="image" value="image" />
            <el-option label="codex" value="codex" />
          </el-select>
        </el-form-item>
        <el-form-item label="调度策略"><el-input v-model="poolForm.dispatch_strategy" /></el-form-item>
        <el-form-item label="粘性 TTL"><el-input-number v-model="poolForm.sticky_ttl_sec" :min="0" /></el-form-item>
        <el-form-item label="描述"><el-input v-model="poolForm.description" type="textarea" :rows="3" /></el-form-item>
        <el-form-item label="启用"><el-switch v-model="poolForm.enabled" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="poolDlg = false">取消</el-button>
        <el-button type="primary" @click="submitPool">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="memberDlg" title="加入账号池" width="560px">
      <el-form label-width="120px">
        <el-form-item label="账号">
          <el-select v-model="memberForm.account_id" filterable style="width: 100%">
            <el-option v-for="a in accounts" :key="a.id" :label="a.email" :value="a.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="权重"><el-input-number v-model="memberForm.weight" :min="1" /></el-form-item>
        <el-form-item label="优先级"><el-input-number v-model="memberForm.priority" :min="1" /></el-form-item>
        <el-form-item label="并发"><el-input-number v-model="memberForm.max_parallel" :min="1" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="memberForm.note" /></el-form-item>
        <el-form-item label="启用"><el-switch v-model="memberForm.enabled" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="memberDlg = false">取消</el-button>
        <el-button type="primary" @click="submitMember">加入</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.split-grid {
  display: grid;
  grid-template-columns: 1.1fr 1fr;
  gap: 16px;
}
.section-title {
  font-weight: 600;
  margin-bottom: 12px;
}
@media (max-width: 1100px) {
  .split-grid {
    grid-template-columns: 1fr;
  }
}
</style>
