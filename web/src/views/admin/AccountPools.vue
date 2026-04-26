<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import * as poolApi from '@/api/account-pools'

const loading = ref(false)
const savingPool = ref(false)
const savingMember = ref(false)
const pools = ref<poolApi.AccountPool[]>([])
const members = ref<poolApi.AccountPoolMember[]>([])
const selectedPoolId = ref<number | null>(null)

const selectedPool = computed(() =>
  pools.value.find((item) => item.id === selectedPoolId.value) || null,
)

const poolDialog = ref(false)
const poolForm = reactive({
  id: 0,
  code: '',
  name: '',
  pool_type: 'image',
  description: '',
  enabled: true,
  dispatch_strategy: 'least_recently_used',
  sticky_ttl_sec: 0,
})

const memberDialog = ref(false)
const memberForm = reactive({
  id: 0,
  account_id: 0,
  enabled: true,
  weight: 100,
  priority: 100,
  max_parallel: 1,
  note: '',
})

async function loadPools(preserveSelection = true) {
  loading.value = true
  try {
    const data = await poolApi.listPools()
    pools.value = data.items || []
    if (!preserveSelection || !selectedPoolId.value || !pools.value.some((item) => item.id === selectedPoolId.value)) {
      selectedPoolId.value = pools.value[0]?.id || null
    }
    await loadMembers()
  } finally {
    loading.value = false
  }
}

async function loadMembers() {
  if (!selectedPoolId.value) {
    members.value = []
    return
  }
  const data = await poolApi.listMembers(selectedPoolId.value)
  members.value = data.items || []
}

function openCreatePool() {
  Object.assign(poolForm, {
    id: 0,
    code: '',
    name: '',
    pool_type: 'image',
    description: '',
    enabled: true,
    dispatch_strategy: 'least_recently_used',
    sticky_ttl_sec: 0,
  })
  poolDialog.value = true
}

function openEditPool(row: poolApi.AccountPool) {
  Object.assign(poolForm, {
    id: row.id,
    code: row.code,
    name: row.name,
    pool_type: row.pool_type,
    description: row.description,
    enabled: row.enabled,
    dispatch_strategy: row.dispatch_strategy,
    sticky_ttl_sec: row.sticky_ttl_sec,
  })
  poolDialog.value = true
}

async function savePool() {
  savingPool.value = true
  try {
    const body = {
      code: poolForm.code,
      name: poolForm.name,
      pool_type: poolForm.pool_type,
      description: poolForm.description,
      enabled: poolForm.enabled,
      dispatch_strategy: poolForm.dispatch_strategy,
      sticky_ttl_sec: poolForm.sticky_ttl_sec,
    }
    if (poolForm.id > 0) {
      await poolApi.updatePool(poolForm.id, body)
      ElMessage.success('账号池已更新')
    } else {
      await poolApi.createPool(body)
      ElMessage.success('账号池已创建')
    }
    poolDialog.value = false
    await loadPools(false)
  } finally {
    savingPool.value = false
  }
}

async function removePool(row: poolApi.AccountPool) {
  await ElMessageBox.confirm(`确认删除账号池「${row.name}」?`, '确认删除', { type: 'warning' })
  await poolApi.deletePool(row.id)
  ElMessage.success('账号池已删除')
  await loadPools(false)
}

function openCreateMember() {
  Object.assign(memberForm, {
    id: 0,
    account_id: 0,
    enabled: true,
    weight: 100,
    priority: 100,
    max_parallel: 1,
    note: '',
  })
  memberDialog.value = true
}

function openEditMember(row: poolApi.AccountPoolMember) {
  Object.assign(memberForm, {
    id: row.id,
    account_id: row.account_id,
    enabled: row.enabled,
    weight: row.weight,
    priority: row.priority,
    max_parallel: row.max_parallel,
    note: row.note,
  })
  memberDialog.value = true
}

async function saveMember() {
  if (!selectedPoolId.value) return
  savingMember.value = true
  try {
    const body = {
      account_id: memberForm.account_id,
      enabled: memberForm.enabled,
      weight: memberForm.weight,
      priority: memberForm.priority,
      max_parallel: memberForm.max_parallel,
      note: memberForm.note,
    }
    if (memberForm.id > 0) {
      await poolApi.updateMember(selectedPoolId.value, memberForm.id, body)
      ElMessage.success('成员已更新')
    } else {
      await poolApi.createMember(selectedPoolId.value, body)
      ElMessage.success('成员已添加')
    }
    memberDialog.value = false
    await loadMembers()
  } finally {
    savingMember.value = false
  }
}

async function removeMember(row: poolApi.AccountPoolMember) {
  if (!selectedPoolId.value) return
  await ElMessageBox.confirm(`确认移除账号 #${row.account_id} ?`, '确认移除', { type: 'warning' })
  await poolApi.deleteMember(selectedPoolId.value, row.id)
  ElMessage.success('成员已移除')
  await loadMembers()
}

onMounted(() => {
  loadPools(false)
})
</script>

<template>
  <div class="page-container" v-loading="loading">
    <div class="page-title">账号池管理</div>
    <div class="page-subtitle">
      Phase 1 语义: <code>priority</code> 越小越优先,同优先级下 <code>weight</code> 越大越靠前;
      <code>max_parallel</code> 当前仍受一号一锁限制。
    </div>

    <el-row :gutter="16" class="pool-layout">
      <el-col :xs="24" :lg="11">
        <div class="card-block">
          <div class="flex-between block-head">
            <div>
              <div class="section-title">账号池</div>
              <div class="section-tip">用于承接图片请求的主池 / fallback 池</div>
            </div>
            <div class="flex-wrap-gap">
              <el-button @click="loadPools()">刷新</el-button>
              <el-button type="primary" @click="openCreatePool()">新建账号池</el-button>
            </div>
          </div>

          <el-table :data="pools" @row-click="(row) => { selectedPoolId = row.id; loadMembers() }" row-key="id">
            <el-table-column prop="id" label="ID" width="70" />
            <el-table-column prop="name" label="名称" min-width="140" />
            <el-table-column prop="code" label="Code" min-width="140" />
            <el-table-column prop="pool_type" label="类型" width="90" />
            <el-table-column label="启用" width="90">
              <template #default="{ row }">
                <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '是' : '否' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="150" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click.stop="openEditPool(row)">编辑</el-button>
                <el-button link type="danger" @click.stop="removePool(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-col>

      <el-col :xs="24" :lg="13">
        <div class="card-block">
          <div class="flex-between block-head">
            <div>
              <div class="section-title">
                {{ selectedPool ? `成员列表 · ${selectedPool.name}` : '成员列表' }}
              </div>
              <div class="section-tip">成员开关只影响当前池,不会影响账号在其他池的可用性。</div>
            </div>
            <div class="flex-wrap-gap">
              <el-button :disabled="!selectedPoolId" @click="loadMembers">刷新</el-button>
              <el-button type="primary" :disabled="!selectedPoolId" @click="openCreateMember">添加成员</el-button>
            </div>
          </div>

          <el-empty v-if="!selectedPoolId" description="请先在左侧选择一个账号池" />
          <el-table v-else :data="members" row-key="id">
            <el-table-column prop="id" label="ID" width="70" />
            <el-table-column prop="account_id" label="账号ID" width="100" />
            <el-table-column label="启用" width="90">
              <template #default="{ row }">
                <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '是' : '否' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="priority" label="Priority" width="100" />
            <el-table-column prop="weight" label="Weight" width="90" />
            <el-table-column prop="max_parallel" label="并发" width="80" />
            <el-table-column prop="note" label="备注" min-width="160" show-overflow-tooltip />
            <el-table-column label="操作" width="150" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="openEditMember(row)">编辑</el-button>
                <el-button link type="danger" @click="removeMember(row)">移除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-col>
    </el-row>

    <el-dialog v-model="poolDialog" :title="poolForm.id ? '编辑账号池' : '新建账号池'" width="560px">
      <el-form label-width="120px">
        <el-form-item label="Code">
          <el-input v-model="poolForm.code" :disabled="poolForm.id > 0" placeholder="例如 image-main" />
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model="poolForm.name" placeholder="例如 主图片池" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="poolForm.pool_type" style="width: 100%">
            <el-option label="image" value="image" />
            <el-option label="mixed" value="mixed" />
          </el-select>
        </el-form-item>
        <el-form-item label="调度策略">
          <el-input v-model="poolForm.dispatch_strategy" placeholder="least_recently_used" />
        </el-form-item>
        <el-form-item label="Sticky TTL">
          <el-input-number v-model="poolForm.sticky_ttl_sec" :min="0" style="width: 100%" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="poolForm.enabled" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="poolForm.description" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="poolDialog = false">取消</el-button>
        <el-button type="primary" :loading="savingPool" @click="savePool">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="memberDialog" :title="memberForm.id ? '编辑成员' : '添加成员'" width="560px">
      <el-form label-width="120px">
        <el-form-item label="账号ID">
          <el-input-number v-model="memberForm.account_id" :disabled="memberForm.id > 0" :min="1" style="width: 100%" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="memberForm.enabled" />
        </el-form-item>
        <el-form-item label="Priority">
          <el-input-number v-model="memberForm.priority" :min="1" style="width: 100%" />
        </el-form-item>
        <el-form-item label="Weight">
          <el-input-number v-model="memberForm.weight" :min="1" style="width: 100%" />
        </el-form-item>
        <el-form-item label="并发">
          <el-input-number v-model="memberForm.max_parallel" :min="1" style="width: 100%" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="memberForm.note" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="memberDialog = false">取消</el-button>
        <el-button type="primary" :loading="savingMember" @click="saveMember">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page-subtitle {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  margin-bottom: 16px;
}
.pool-layout {
  align-items: stretch;
}
.block-head {
  margin-bottom: 12px;
}
.section-title {
  font-size: 16px;
  font-weight: 600;
}
.section-tip {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  margin-top: 4px;
}
</style>
